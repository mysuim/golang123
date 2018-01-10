package collect

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/shen100/golang123/controller/common"
	"github.com/shen100/golang123/model"
)

/*
// CreateCollect 收藏文章或收藏投票
func CreateCollect(ctx iris.Context) {
	SendErrJSON := common.SendErrJSON
	var collect model.Collect
	var article model.Article
	var vote model.Vote

	if err := ctx.ReadJSON(&collect); err != nil {
		fmt.Println(err.Error())
		SendErrJSON("参数无效", ctx)
		return
	}

	if collect.SourceName != model.CollectSourceArticle && collect.SourceName != model.CollectSourceVote {
		SendErrJSON("sourceName无效", ctx)
		return
	}

	if collect.SourceName == model.CollectSourceArticle {
		if err := model.DB.First(&article, collect.SourceID).Error; err != nil {
			SendErrJSON("无效的sourceID", ctx)
			return
		}
	}

	if collect.SourceName == model.CollectSourceVote {
		if err := model.DB.First(&vote, collect.SourceID).Error; err != nil {
			SendErrJSON("无效的sourceID", ctx)
			return
		}
	}

	if err := model.DB.First(&collect.Folder, collect.FolderID).Error; err != nil {
		SendErrJSON("无效的收藏夹id", ctx)
		return
	}

	var collects []model.Collect
	if err := model.DB.Where("source_id=? AND source_name=?", collect.SourceID, collect.SourceName).
		Preload("Folder").Find(&collects).Error; err == nil {
		for i := 0; i < len(collects); i++ {
			if collects[i].FolderID == collect.FolderID {
				// 在相同的收藏夹下，之前已经收藏过
				SendErrJSON("之前已经收藏过", ctx)
				return
			}
		}
	}

	user, _ := manager.Sess.Start(ctx).Get("user").(model.User)
	collect.UserID = user.ID

	tx := model.DB.Begin()

	if err := tx.Save(&collect).Error; err != nil {
		fmt.Println(err.Error())
		tx.Rollback()
		SendErrJSON("error", ctx)
		return
	}

	if err := tx.Model(&user).Update("collect_count", user.CollectCount+1).Error; err != nil {
		fmt.Println(err.Error())
		tx.Rollback()
		SendErrJSON("error", ctx)
		return
	}
	manager.Sess.Start(ctx).Set("user", user)

	if collect.SourceName == model.CollectSourceArticle {
		if err := tx.Model(&article).Update("collect_count", article.CollectCount+1).Error; err != nil {
			fmt.Println(err.Error())
			tx.Rollback()
			SendErrJSON("error", ctx)
			return
		}
		if err := tx.Model(&article).Related(&article.User).Error; err != nil {
			SendErrJSON("error", ctx)
			tx.Rollback()
			return
		}
		// 自己收藏自己的话题，积分不增加
		if article.User.ID != user.ID {
			if err := tx.Model(&article.User).Update("score", article.User.Score+model.ByCollectScore).Error; err != nil {
				fmt.Println(err.Error())
				tx.Rollback()
				SendErrJSON("error", ctx)
				return
			}
		}
	}

	if collect.SourceName == model.CollectSourceVote {
		if err := tx.Model(&vote).Update("collect_count", vote.CollectCount+1).Error; err != nil {
			fmt.Println(err.Error())
			tx.Rollback()
			SendErrJSON("error", ctx)
			return
		}
		if err := tx.Model(&vote).Related(&vote.User).Error; err != nil {
			SendErrJSON("error", ctx)
			tx.Rollback()
			return
		}
		// 自己收藏自己的投票，积分不增加
		if vote.User.ID != user.ID {
			if err := tx.Model(&vote.User).Update("score", vote.User.Score+model.ByCollectScore).Error; err != nil {
				fmt.Println(err.Error())
				tx.Rollback()
				SendErrJSON("error", ctx)
				return
			}
		}
	}

	tx.Commit()

	ctx.JSON(iris.Map{
		"errNo": model.ErrorCode.SUCCESS,
		"msg":   "success",
		"data":  collect,
	})
}

// DeleteCollect 删除收藏
func DeleteCollect(ctx iris.Context) {
	SendErrJSON := common.SendErrJSON
	var collect model.Collect
	collectID, idErr := ctx.Params().GetInt("id")
	if idErr != nil {
		SendErrJSON("无效的ID", ctx)
		return
	}

	if err := model.DB.First(&collect, collectID).Error; err != nil {
		SendErrJSON("无效的id", ctx)
		return
	}
	if err := model.DB.Delete(&collect).Error; err != nil {
		SendErrJSON("error", ctx)
		return
	}

	user, _ := manager.Sess.Start(ctx).Get("user").(model.User)

	tx := model.DB.Begin()
	if err := tx.Model(&user).Update("collect_count", user.CollectCount-1).Error; err != nil {
		fmt.Println(err.Error())
		tx.Rollback()
		SendErrJSON("error", ctx)
		return
	}
	manager.Sess.Start(ctx).Set("user", user)

	// 删除收藏后，相关积分保持不变
	if collect.SourceName == model.CollectSourceArticle {
		var article model.Article
		if err := tx.First(&article, collect.SourceID).Error; err != nil {
			tx.Rollback()
			SendErrJSON("error", ctx)
			return
		}
		if err := tx.Model(&article).Update("collect_count", article.CollectCount-1).Error; err != nil {
			fmt.Println(err.Error())
			tx.Rollback()
			SendErrJSON("error", ctx)
			return
		}
	}

	if collect.SourceName == model.CollectSourceVote {
		var vote model.Vote
		if err := tx.First(&vote, collect.SourceID).Error; err != nil {
			tx.Rollback()
			SendErrJSON("error", ctx)
			return
		}
		if err := tx.Model(&vote).Update("collect_count", vote.CollectCount-1).Error; err != nil {
			fmt.Println(err.Error())
			tx.Rollback()
			SendErrJSON("error", ctx)
			return
		}
	}

	tx.Commit()
	ctx.JSON(iris.Map{
		"errNo": model.ErrorCode.SUCCESS,
		"msg":   "success",
		"data": iris.Map{
			"id": collect.ID,
		},
	})
}

// Collects 根据收藏夹查询用户已收藏的话题或投票
func Collects(ctx iris.Context) {
	SendErrJSON := common.SendErrJSON
	var collects []model.Collect

	userID, userIDErr := strconv.Atoi(ctx.FormValue("userID"))
	if userIDErr != nil {
		SendErrJSON("无效的userID", ctx)
		return
	}
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		SendErrJSON("无效的userID", ctx)
		return
	}

	pageNo, pageNoErr := strconv.Atoi(ctx.FormValue("pageNo"))

	if pageNoErr != nil || pageNo < 1 {
		pageNo = 1
	}

	folderID, folderIDErr := strconv.Atoi(ctx.FormValue("folderID"))
	if folderIDErr != nil {
		SendErrJSON("无效的folderID", ctx)
		return
	}

	var folder model.Folder
	if err := model.DB.First(&folder, folderID).Error; err != nil {
		SendErrJSON("无效的folderID", ctx)
		return
	}

	var pageSize int
	var pageSizeErr error
	if pageSize, pageSizeErr = strconv.Atoi(ctx.FormValue("pageSize")); pageSizeErr != nil {
		SendErrJSON("无效的pageSize", ctx)
		return
	}

	if pageSize < 1 || pageSize > model.MaxPageSize {
		SendErrJSON("无效的pageSize", ctx)
		return
	}

	offset := (pageNo - 1) * pageSize

	if err := model.DB.Where("folder_id=? AND user_id=?", folderID, userID).Offset(offset).
		Limit(pageSize).Order("created_at DESC").Find(&collects).Error; err != nil {
		fmt.Println(err.Error())
		SendErrJSON("error", ctx)
		return
	}

	var totalCount int
	if err := model.DB.Model(&model.Collect{}).Where("folder_id=? AND user_id=?", folderID, userID).
		Count(&totalCount).Error; err != nil {
		fmt.Println(err.Error())
		SendErrJSON("error", ctx)
		return
	}

	var results []map[string]interface{}
	for i := 0; i < len(collects); i++ {
		data := make(map[string]interface{})
		var article model.Article
		var vote model.Vote
		data["id"] = collects[i].ID
		if collects[i].SourceName == model.CollectSourceArticle {
			if err := model.DB.Model(&collects[i]).Related(&article, "articles", "source_id").Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					fmt.Println(err.Error())
					SendErrJSON("error", ctx)
					return
				}
			}
			data["sourceName"] = model.CollectSourceArticle
			data["articleID"] = article.ID
			data["articleName"] = article.Name
			data["content"] = utils.MarkdownToHTML(article.Content)
		} else if collects[i].SourceName == model.CollectSourceVote {
			if err := model.DB.Model(&collects[i]).Related(&vote, "votes", "source_id").Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					fmt.Println(err.Error())
					SendErrJSON("error", ctx)
					return
				}
			}
			data["sourceName"] = model.CollectSourceVote
			data["voteID"] = vote.ID
			data["voteName"] = vote.Name
			data["content"] = utils.MarkdownToHTML(vote.Content)
		}
		results = append(results, data)
	}

	ctx.JSON(iris.Map{
		"errNo": model.ErrorCode.SUCCESS,
		"msg":   "success",
		"data": iris.Map{
			"folderID":   folder.ID,
			"folderName": folder.Name,
			"collects":   results,
			"pageNo":     pageNo,
			"pageSize":   pageSize,
			"totalCount": totalCount,
		},
	})
}

// CreateFolder 创建收藏夹
func CreateFolder(ctx iris.Context) {
	SendErrJSON := common.SendErrJSON
	var folder model.Folder
	if err := ctx.ReadJSON(&folder); err != nil {
		SendErrJSON("参数无效", ctx)
		return
	}

	folder.Name = strings.TrimSpace(folder.Name)
	folder.Name = bluemonday.UGCPolicy().Sanitize(folder.Name)

	if folder.Name == "" {
		SendErrJSON("收藏夹名称不能为空", ctx)
		return
	}

	if utf8.RuneCountInString(folder.Name) > model.MaxNameLen {
		SendErrJSON("收藏夹名称不能超过"+fmt.Sprintf("%d", model.MaxNameLen)+"个字符", ctx)
		return
	}

	if folder.ParentID != model.NoParent {
		var parentFolder model.Folder
		if err := model.DB.First(&parentFolder, folder.ParentID).Error; err != nil {
			SendErrJSON("无效的parentID", ctx)
			return
		}
	}

	user := manager.Sess.Start(ctx).Get("user").(model.User)

	var folders []model.Folder
	var queryFoldersErr error
	if folders, queryFoldersErr = queryFolders(int(user.ID)); queryFoldersErr != nil {
		SendErrJSON("error", ctx)
		return
	}

	if len(folders) >= model.MaxFolderCount {
		msg := "最多只能创建" + fmt.Sprintf("%d", model.MaxFolderCount) + "个收藏夹"
		SendErrJSON(msg, ctx)
		return
	}

	folder.UserID = user.ID

	if err := model.DB.Create(&folder).Error; err != nil {
		SendErrJSON("error", ctx)
		return
	}

	ctx.JSON(iris.Map{
		"errNo": model.ErrorCode.SUCCESS,
		"msg":   "success",
		"data": iris.Map{
			"id":       folder.ID,
			"name":     folder.Name,
			"collects": nil,
		},
	})
}
*/

func queryFolders(userID int) ([]model.Folder, error) {
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}

	var folders []model.Folder
	if err := model.DB.Where("user_id=?", userID).Order("created_at DESC").Find(&folders).Error; err != nil {
		return nil, err
	}
	return folders, nil
}

// Folders 查询用户的收藏夹列表
func Folders(c *gin.Context) {
	SendErrJSON := common.SendErrJSON
	userID, userIDErr := strconv.Atoi(c.Param("userID"))
	if userIDErr != nil {
		SendErrJSON("无效的userID", c)
		return
	}

	var folders []model.Folder
	var err error
	if folders, err = queryFolders(userID); err != nil {
		SendErrJSON("error", c)
		return
	}
	var results []map[string]interface{}
	for i := 0; i < len(folders); i++ {
		var data = map[string]interface{}{
			"id":        folders[i].ID,
			"createdAt": folders[i].CreatedAt,
			"updatedAt": folders[i].UpdatedAt,
			"deletedAt": folders[i].DeletedAt,
			"name":      folders[i].Name,
			"userID":    folders[i].UserID,
			"parentID":  folders[i].ParentID,
		}
		var collectCount uint
		if err := model.DB.Model(&model.Collect{}).Where("folder_id = ?", folders[i].ID).Count(&collectCount).Error; err != nil {
			SendErrJSON("error", c)
			return
		}
		data["collectCount"] = collectCount
		results = append(results, data)
	}
	c.JSON(http.StatusOK, gin.H{
		"errNo": model.ErrorCode.SUCCESS,
		"msg":   "success",
		"data": gin.H{
			"folders": results,
		},
	})
}

// FoldersWithSource 查询用户的收藏夹列表，并且返回每个收藏夹中收藏了哪些话题或投票
func FoldersWithSource(c *gin.Context) {
	SendErrJSON := common.SendErrJSON
	iuser, exists := c.Get("user")

	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"errNo": model.ErrorCode.SUCCESS,
			"msg":   "success",
			"data": gin.H{
				"folders": make([]interface{}, 0),
			},
		})
		return
	}

	user := iuser.(model.User)
	var folders []model.Folder
	var queryFoldersErr error
	if folders, queryFoldersErr = queryFolders(int(user.ID)); queryFoldersErr != nil {
		fmt.Println(queryFoldersErr.Error())
		SendErrJSON("error", c)
		return
	}

	var results []interface{}
	for i := 0; i < len(folders); i++ {
		var collects []model.Collect
		if err := model.DB.Where("folder_id=?", folders[i].ID).Find(&collects).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				fmt.Println(err.Error())
				SendErrJSON("error", c)
				return
			}
		}
		results = append(results, gin.H{
			"id":       folders[i].ID,
			"name":     folders[i].Name,
			"collects": collects,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"errNo": model.ErrorCode.SUCCESS,
		"msg":   "success",
		"data": gin.H{
			"folders": results,
		},
	})
}
