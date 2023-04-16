package main

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/KNN3-Network/snapshot/utils"
	"github.com/machinebox/graphql"
	"go.uber.org/zap"
)

type StringSlice []string

var logger = utils.Logger

func (ss *StringSlice) Scan(value interface{}) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, &ss)
	case string:
		return json.Unmarshal([]byte(v), &ss)
	default:
		return fmt.Errorf("unsupported Scan destination")
	}
}

func (ss StringSlice) Value() (driver.Value, error) {
	if len(ss) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(ss)
}

type Vote struct {
	ID              string      `json:"id"`
	Voter           string      `json:"voter" gorm:"index:idx_voter"`
	Choice          int         `json:"choice"`
	SpaceID         string      `json:"spaceId" gorm:"column:spaceId;index:idx_space_id"`
	SpaceName       string      `json:"spaceName" gorm:"column:spaceName;index:idx_space_name"`
	SpaceAvatar     string      `json:"spaceAvatar" gorm:"column:spaceAvatar"`
	SpaceAdmins     StringSlice `json:"spaceAdmins" gorm:"column:spaceAdmins;type:json"`
	SpaceModerators StringSlice `json:"spaceModerators" gorm:"column:spaceModerators;type:json"`
	SpaceMembers    StringSlice `json:"spaceMembers" gorm:"column:spaceMembers;type:json"`
	ProposalID      string      `json:"proposalId" gorm:"column:proposalId;index:idx_proposal_id"`
	ProposalAuthor  string      `json:"proposalAuthor" gorm:"column:proposalAuthor"`
	ProposalTitle   string      `json:"proposalTitle" gorm:"column:proposalTitle"`
	Created         time.Time   `json:"created" gorm:"index:idx_created;type:timestamp"`
}

func (Vote) TableName() string {
	return "vote"
}

func toStringSlice(i interface{}) []string {
	if i == nil {
		return nil
	}
	slice := i.([]interface{})
	result := make([]string, len(slice))
	for j, v := range slice {
		result[j] = v.(string)
	}
	return result
}

func queryVotes(createdGt int64) ([]Vote, error) {
	logger.Info("查询时间", zap.Int64("created_timestamp", createdGt))
	query := `
        query Votes($createGt: Int!) {
            votes(first: 1000, where: { created_gt:$createGt }, orderBy: "created", orderDirection: asc) {
                id
                voter
				choice
                created
                space {
					id
					name
					avatar
					admins
					moderators
					members	
                }
                proposal {
					author
                    id
                    title
                }
            }
        }
    `
	client := graphql.NewClient("https://hub.snapshot.org/graphql")
	req := graphql.NewRequest(query)
	req.Var("createGt", createdGt)
	var responseData map[string]interface{}
	if err := client.Run(context.Background(), req, &responseData); err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	votes := responseData["votes"].([]interface{})
	var result []Vote // 声明一个空的 Vote 切片
	for _, v := range votes {
		vote := v.(map[string]interface{})
		space := vote["space"].(map[string]interface{})
		admins := toStringSlice(space["admins"])
		moderators := toStringSlice(space["moderators"])
		members := toStringSlice(space["members"])
		var proposalID, proposalTitle string
		if proposal, ok := vote["proposal"].(map[string]interface{}); ok {
			proposalID = proposal["id"].(string)
			proposalTitle = proposal["title"].(string)
		}
		spaceName, ok := space["name"].(string)
		if !ok {
			spaceName = ""
		}
		result = append(result, Vote{
			ID:              vote["id"].(string),
			Voter:           vote["voter"].(string),
			Choice:          int(vote["choice"].(float64)),
			Created:         time.Unix(int64(vote["created"].(float64)), 0),
			SpaceID:         space["id"].(string),
			SpaceName:       spaceName,
			SpaceAvatar:     space["avatar"].(string),
			SpaceAdmins:     admins,
			SpaceMembers:    members,
			SpaceModerators: moderators,
			ProposalID:      proposalID,
			ProposalTitle:   proposalTitle,
		})
	}

	return result, nil
}

func main() {
	// 获取数据库连接
	db := utils.GetDB()

	for {
		var vote Vote
		err := db.Order("created DESC").First(&vote).Error
		if err != nil {
			logger.Error(err.Error())
		}
		createGt := int64(0)
		if vote.ID != "" {
			createGt = vote.Created.Unix()
		}
		result, err := queryVotes(createGt)
		if err != nil {
			logger.Error(err.Error())
			break
		}
		db.Create(&result)
		logger.Info("插入成功", zap.Any("created", createGt))
	}

}
