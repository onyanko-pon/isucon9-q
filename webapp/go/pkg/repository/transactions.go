package repository

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/onyanko-pon/isucon9-q/pkg/model"
)

type ItemDetailEntity struct {
	ID       int64
	SellerID int64
	// Seller                    *model.UserSimple
	SellerAccoutName   string
	SellerNumSellItems int
	BuyerID            int64
	BuyerAccoutName    string
	BuyerNumSellItems  int
	// Buyer                     *model.UserSimple
	Status      string
	Name        string
	Price       int
	Description string
	ImageName   string

	CategoryID       int
	CategoryParentID int
	CategoryName     string
	// Category                  *model.Category
	TransactionEvidenceID     int64
	TransactionEvidenceStatus string
	ShippingStatus            string
	CreatedAt                 int64
}

const (
	ItemStatusOnSale    = "on_sale"
	ItemStatusTrading   = "trading"
	ItemStatusSoldOut   = "sold_out"
	ItemStatusStop      = "stop"
	ItemStatusCancel    = "cancel"
	TransactionsPerPage = 10
)

func GetTransactions(dbx *sqlx.DB, user model.User, itemID int64, createdAt int64) ([]model.ItemDetail, error) {
	items := []ItemDetailEntity{}

	querySelect := "SELECT " +
		"items.id as id " +
		"sellers.id as seller_id, sellers.accout_name as seller_accout_name, sellers.num_sell_items as seller_num_sell_items " +
		"buyers.id as buyer_id, buyers.accout_name as buyer_accout_name, buyers.num_sell_items as buyer_num_sell_items " +
		"items.status as status, items.name as name, items.price as price, items.descriptions as descriptions, items.image_name as image_name " +
		"categories.id as category_id, categories.parent_id as category_parent_id, categories.name category_name " +
		"transaction_evidences.id as transaction_evidence_id, transaction_evidences.status as transaction_evidence_status" +
		"shippings.status as shipping_status, items.created_at as created_at FROM `items` "

	queryJoin := " " +
		"JOIN users as sellers ON sellers.id = items.seller_id " +
		"JOIN users as buyers ON buyers.id = items.buyer_id " +
		"JOIN categories ON categories.id = items.category_id " +
		"JOIN transaction_evidences on transaction_evidences.item_id = items.id " +
		"JOIN shippings on shippings.item_id = items.id "

	tx := dbx.MustBegin()
	if itemID > 0 && createdAt > 0 {

		queryWhere := "WHERE (`items.seller_id` = ? OR `items.buyer_id` = ?) AND `items.status` IN (?,?,?,?,?) AND (`items.created_at` < ?  OR (`items.created_at` <= ? AND `items.id` < ?)) "

		query := querySelect + queryJoin + queryWhere + " ORDER BY `items.created_at` DESC, `id` GroupBy items.id DESC LIMIT ?"
		err := tx.Select(&items,
			query,
			user.ID,
			user.ID,
			ItemStatusOnSale,
			ItemStatusTrading,
			ItemStatusSoldOut,
			ItemStatusCancel,
			ItemStatusStop,
			time.Unix(createdAt, 0),
			time.Unix(createdAt, 0),
			itemID,
			TransactionsPerPage+1,
		)
		if err != nil {
			log.Print(err)
			// outputErrorMsg(w, http.StatusInternalServerError, "db error")
			tx.Rollback()
			return nil, err
		}
	} else {
		// 1st page
		queryWhere := "WHERE (`seller_id` = ? OR `buyer_id` = ?) AND `status` IN (?,?,?,?,?) "
		query := querySelect + queryJoin + queryWhere + " ORDER BY `items.created_at` DESC, `id` GroupBy items.id DESC LIMIT ?"

		err := tx.Select(&items,
			query,
			user.ID,
			user.ID,
			ItemStatusOnSale,
			ItemStatusTrading,
			ItemStatusSoldOut,
			ItemStatusCancel,
			ItemStatusStop,
			TransactionsPerPage+1,
		)
		if err != nil {
			log.Print(err)
			// outputErrorMsg(w, http.StatusInternalServerError, "db error")
			tx.Rollback()
			return nil, err
		}
	}
	tx.Commit()

	itemDetails := []model.ItemDetail{}
	for _, item := range items {
		category := model.GetCategoryByID(item.CategoryID)
		d := model.ItemDetail{
			ID:       item.ID,
			SellerID: item.SellerID,
			Seller: &model.UserSimple{
				ID:           item.SellerID,
				AccountName:  item.SellerAccoutName,
				NumSellItems: item.SellerNumSellItems,
			},
			BuyerID: item.BuyerID,
			Buyer: &model.UserSimple{
				ID:           item.BuyerID,
				AccountName:  item.BuyerAccoutName,
				NumSellItems: item.BuyerNumSellItems,
			},
			Status:                    item.Status,
			Name:                      item.Name,
			Price:                     item.Price,
			Description:               item.Description,
			ImageURL:                  fmt.Sprintf("/upload/%s", item.ImageName),
			CategoryID:                item.CategoryID,
			Category:                  &category,
			TransactionEvidenceID:     item.TransactionEvidenceID,
			TransactionEvidenceStatus: item.TransactionEvidenceStatus,
			ShippingStatus:            item.ShippingStatus,
			CreatedAt:                 item.CreatedAt,
		}

		itemDetails = append(itemDetails, d)
	}
	return itemDetails, nil
}
