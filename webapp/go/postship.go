package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type TransactionEvidenceEntity struct {
	ID                 int64     `json:"id" db:"id"`
	SellerID           int64     `json:"seller_id" db:"seller_id"`
	BuyerID            int64     `json:"buyer_id" db:"buyer_id"`
	Status             string    `json:"status" db:"status"`
	ItemID             int64     `json:"item_id" db:"item_id"`
	ItemName           string    `json:"item_name" db:"item_name"`
	ItemPrice          int       `json:"item_price" db:"item_price"`
	ItemDescription    string    `json:"item_description" db:"item_description"`
	ItemCategoryID     int       `json:"item_category_id" db:"item_category_id"`
	ItemRootCategoryID int       `json:"item_root_category_id" db:"item_root_category_id"`
	CreatedAt          time.Time `json:"-" db:"created_at"`
	UpdatedAt          time.Time `json:"-" db:"updated_at"`

	ItemStatus        string `db:"item_status"`
	ShippingStatus    string `db:"shipping_status"`
	ShippingReserveID string `db:"shipping_reserve_id"`
}

func postShip(w http.ResponseWriter, r *http.Request) {
	reqps := reqPostShip{}

	err := json.NewDecoder(r.Body).Decode(&reqps)
	if err != nil {
		outputErrorMsg(w, http.StatusBadRequest, "json decode error")
		return
	}

	csrfToken := reqps.CSRFToken
	itemID := reqps.ItemID

	if csrfToken != getCSRFToken(r) {
		outputErrorMsg(w, http.StatusUnprocessableEntity, "csrf token error")

		return
	}

	seller, errCode, errMsg := getUser(r)
	if errMsg != "" {
		outputErrorMsg(w, errCode, errMsg)
		return
	}

	transactionEvidence := TransactionEvidenceEntity{}

	querySelect := "SELECT transaction_evidences.*, items.status as item_status, shippings.status as shipping_status, shippings.reserve_id as shipping_reserve_id FROM transaction_evidences "
	queryJoin := "LEFT JOIN items ON items.id = transaction_evidences.item_id " +
		"LEFT JOIN shippings ON shippings.transaction_evidence_id = transaction_evidences.id "
	queryWhere := "WHERE item_id = ? "
	query := querySelect + queryJoin + queryWhere

	err = dbx.Get(&transactionEvidence, query, itemID)
	if err == sql.ErrNoRows {
		outputErrorMsg(w, http.StatusNotFound, "transaction_evidences not found")
		return
	}
	if err != nil {
		log.Print(err)
		outputErrorMsg(w, http.StatusInternalServerError, "db error")

		return
	}

	if transactionEvidence.SellerID != seller.ID {
		outputErrorMsg(w, http.StatusForbidden, "権限がありません")
		return
	}

	// item := model.Item{}
	// err = dbx.Get(&item, "SELECT * FROM `items` WHERE `id` = ?", itemID)
	// if err == sql.ErrNoRows {
	// 	outputErrorMsg(w, http.StatusNotFound, "item not found")
	// 	return
	// }
	// if err != nil {
	// 	log.Print(err)
	// 	outputErrorMsg(w, http.StatusInternalServerError, "db error")
	// 	return
	// }

	if transactionEvidence.ItemStatus != ItemStatusTrading {
		outputErrorMsg(w, http.StatusForbidden, "商品が取引中ではありません")
		return
	}

	// err = dbx.Get(&transactionEvidence, "SELECT * FROM `transaction_evidences` WHERE `id` = ?", transactionEvidence.ID)
	// if err == sql.ErrNoRows {
	// 	outputErrorMsg(w, http.StatusNotFound, "transaction_evidences not found")
	// 	return
	// }
	// if err != nil {
	// 	log.Print(err)
	// 	outputErrorMsg(w, http.StatusInternalServerError, "db error")
	// 	return
	// }

	if transactionEvidence.Status != TransactionEvidenceStatusWaitShipping {
		outputErrorMsg(w, http.StatusForbidden, "準備ができていません")
		return
	}

	// shipping := model.Shipping{}
	// err = dbx.Get(&shipping, "SELECT * FROM `shippings` WHERE `transaction_evidence_id` = ?", transactionEvidence.ID)
	// if err == sql.ErrNoRows {
	// 	outputErrorMsg(w, http.StatusNotFound, "shippings not found")
	// 	return
	// }
	if transactionEvidence.ShippingStatus == "" {
		outputErrorMsg(w, http.StatusNotFound, "shippings not found")
		return
	}
	// if err != nil {
	// 	log.Print(err)
	// 	outputErrorMsg(w, http.StatusInternalServerError, "db error")
	// 	return
	// }

	tx := dbx.MustBegin()
	img, err := APIShipmentRequest(getShipmentServiceURL(), &APIShipmentRequestReq{
		ReserveID: transactionEvidence.ShippingReserveID,
	})
	if err != nil {
		log.Print(err)
		outputErrorMsg(w, http.StatusInternalServerError, "failed to request to shipment service")
		tx.Rollback()

		return
	}

	_, err = tx.Exec("UPDATE `shippings` SET `status` = ?, `img_binary` = ?, `updated_at` = ? WHERE `transaction_evidence_id` = ?",
		ShippingsStatusWaitPickup,
		img,
		time.Now(),
		transactionEvidence.ID,
	)
	if err != nil {
		log.Print(err)

		outputErrorMsg(w, http.StatusInternalServerError, "db error")
		tx.Rollback()
		return
	}

	tx.Commit()

	rps := resPostShip{
		Path:      fmt.Sprintf("/transactions/%d.png", transactionEvidence.ID),
		ReserveID: transactionEvidence.ShippingReserveID,
	}
	json.NewEncoder(w).Encode(rps)
}
