package repository

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/onyanko-pon/isucon9-q/pkg/model"
)

type ItemDetailEntity struct {
	ID       int64 `db:"id"`
	SellerID int64 `db:"seller_id"`
	// Seller                    *model.UserSimple
	SellerAccountName  string  `db:"seller_account_name"`
	SellerNumSellItems int     `db:"seller_num_sell_items"`
	BuyerID            *int64  `db:"buyer_id"`
	BuyerAccountName   *string `db:"buyer_account_name"`
	BuyerNumSellItems  *int    `db:"buyer_num_sell_items"`
	// Buyer                     *model.UserSimple
	Status      string `db:"status"`
	Name        string `db:"name"`
	Price       int    `db:"price"`
	Description string `db:"description"`
	ImageName   string `db:"image_name"`

	CategoryID int `db:"category_id"`
	// CategoryParentID int    `db:"category_parent_id"`
	// CategoryName     string `db:"category_name"`
	// Category                  *model.Category
	TransactionEvidenceID     *int64    `db:"transaction_evidence_id"`
	TransactionEvidenceStatus *string   `db:"transaction_evidence_status"`
	ShippingStatus            *string   `db:"shipping_status"`
	CreatedAt                 time.Time `db:"created_at"`
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
		"items.id as id, " +
		"sellers.id as seller_id, sellers.account_name as seller_account_name, sellers.num_sell_items as seller_num_sell_items, " +
		"buyers.id as buyer_id, buyers.account_name as buyer_account_name, buyers.num_sell_items as buyer_num_sell_items, " +
		"items.status as status, items.name as name, items.price as price, items.description as description, items.image_name as image_name, " +
		"items.category_id as category_id, " +
		// "categories.id as category_id, categories.parent_id as category_parent_id, categories.category_name category_name, " +
		"transaction_evidences.id as transaction_evidence_id, transaction_evidences.status as transaction_evidence_status, " +
		"shippings.status as shipping_status, items.created_at as created_at FROM items "

	queryJoin := " " +
		"LEFT JOIN users as sellers ON sellers.id = items.seller_id " +
		"LEFT JOIN users as buyers ON buyers.id = items.buyer_id " +
		// "LEFT JOIN categories ON categories.id = items.category_id " +
		"LEFT JOIN transaction_evidences on transaction_evidences.item_id = items.id " +
		"LEFT JOIN shippings on shippings.item_id = items.id "

	if itemID > 0 && createdAt > 0 {

		// queryWhere := "WHERE (items.seller_id = ? OR items.buyer_id = ?) AND items.status IN (?,?,?,?,?) AND (items.created_at < ?  OR (items.created_at <= ? AND items.id < ?)) "
		queryWhere := "WHERE (items.seller_id = ? OR items.buyer_id = ?) AND (items.created_at < ?  OR (items.created_at <= ? AND items.id < ?)) "

		query := querySelect + queryJoin + queryWhere + " Group By items.id ORDER BY items.created_at DESC, items.id DESC LIMIT ?"
		err := dbx.Select(&items,
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
			return nil, err
		}
	} else {
		// 1st page
		// queryWhere := "WHERE (items.seller_id = ? OR items.buyer_id = ?) AND items.status IN (?,?,?,?,?) "
		queryWhere := "WHERE (items.seller_id = ? OR items.buyer_id = ?) "
		query := querySelect + queryJoin + queryWhere + " Group By items.id  ORDER BY items.created_at DESC, items.id DESC LIMIT ?"

		// fmt.Println(query)

		err := dbx.Select(&items,
			query,
			user.ID,
			user.ID,
			// ItemStatusOnSale,
			// ItemStatusTrading,
			// ItemStatusSoldOut,
			// ItemStatusCancel,
			// ItemStatusStop,
			TransactionsPerPage+1,
		)
		if err != nil {
			log.Print(err)
			// outputErrorMsg(w, http.StatusInternalServerError, "db error")
			return nil, err
		}
	}

	itemDetails := []model.ItemDetail{}
	for _, item := range items {
		category := model.GetCategoryByID(item.CategoryID)
		var buyerid int64
		var buyer *model.UserSimple
		if item.BuyerID != nil {
			buyerid = *(item.BuyerID)
			buyer = &model.UserSimple{
				ID:           buyerid,
				AccountName:  *item.BuyerAccountName,
				NumSellItems: *item.BuyerNumSellItems,
			}
		} else {
			buyerid = 0
		}

		var eid int64
		var estatus string
		var sstatus string
		if item.TransactionEvidenceID != nil {
			eid = *item.TransactionEvidenceID
			estatus = *item.TransactionEvidenceStatus
			sstatus = *item.ShippingStatus
		}

		d := model.ItemDetail{
			ID:       item.ID,
			SellerID: item.SellerID,
			Seller: &model.UserSimple{
				ID:           item.SellerID,
				AccountName:  item.SellerAccountName,
				NumSellItems: item.SellerNumSellItems,
			},
			BuyerID:                   buyerid,
			Buyer:                     buyer,
			Status:                    item.Status,
			Name:                      item.Name,
			Price:                     item.Price,
			Description:               item.Description,
			ImageURL:                  fmt.Sprintf("/upload/%s", item.ImageName),
			CategoryID:                item.CategoryID,
			Category:                  &category,
			TransactionEvidenceID:     eid,
			TransactionEvidenceStatus: estatus,
			ShippingStatus:            sstatus,
			CreatedAt:                 item.CreatedAt.Unix(),
		}

		itemDetails = append(itemDetails, d)
	}
	return itemDetails, nil
}
