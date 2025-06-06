package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type PublicContract struct {
	contractapi.Contract
}

type PublicDocument struct {
	DocumentID  string `json:"documentId"`
	Institution string `json:"institution"`
	UserID      string `json:"userId"`
}

type AuditLog struct {
	TxID        string `json:"txID"`
	DocumentID  string `json:"documentId"`
	Institution string `json:"institution"`
	UserID      string `json:"userId"`
	Operation   string `json:"operation"`
	Timestamp   string `json:"timestamp"`
}

func (c *PublicContract) RegisterDocument(ctx contractapi.TransactionContextInterface, documentId, institution, userId string) error {
	doc := PublicDocument{
		DocumentID:  documentId,
		Institution: institution,
		UserID:      userId,
	}
	data, _ := json.Marshal(doc)
	if err := ctx.GetStub().PutState(documentId, data); err != nil {
		return err
	}

	txID := ctx.GetStub().GetTxID()
	ts, _ := ctx.GetStub().GetTxTimestamp()
	timestamp := time.Unix(ts.Seconds, int64(ts.Nanos)).Format(time.RFC3339)
	log := AuditLog{
		TxID:        txID,
		DocumentID:  documentId,
		Institution: institution,
		UserID:      userId,
		Operation:   "create",
		Timestamp:   timestamp,
	}
	logData, _ := json.Marshal(log)
	logKey := "AUDIT_" + txID
	return ctx.GetStub().PutState(logKey, logData)
}

func (c *PublicContract) GetDocumentByID(ctx contractapi.TransactionContextInterface, documentId string) (*PublicDocument, error) {
	data, err := ctx.GetStub().GetState(documentId)
	if err != nil || data == nil {
		return nil, fmt.Errorf("document not found")
	}
	var doc PublicDocument
	_ = json.Unmarshal(data, &doc)
	return &doc, nil
}

func (c *PublicContract) QueryByInstitution(ctx contractapi.TransactionContextInterface, institution string) ([]*PublicDocument, error) {
	results := []*PublicDocument{}
	it, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer it.Close()

	for it.HasNext() {
		item, _ := it.Next()
		var doc PublicDocument
		if json.Unmarshal(item.Value, &doc) == nil && doc.Institution == institution {
			results = append(results, &doc)
		}
	}
	return results, nil
}

func (c *PublicContract) QueryByUser(ctx contractapi.TransactionContextInterface, userId string) ([]*PublicDocument, error) {
	results := []*PublicDocument{}
	it, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer it.Close()

	for it.HasNext() {
		item, _ := it.Next()
		var doc PublicDocument
		if json.Unmarshal(item.Value, &doc) == nil && doc.UserID == userId {
			results = append(results, &doc)
		}
	}
	return results, nil
}

func (c *PublicContract) QueryAuditLogs(ctx contractapi.TransactionContextInterface, filterType, filterValue, startDate, endDate string) ([]*AuditLog, error) {
	results := []*AuditLog{}
	it, err := ctx.GetStub().GetStateByRange("AUDIT_", "AUDIT_z")
	if err != nil {
		return nil, err
	}
	defer it.Close()

	start, _ := time.Parse(time.RFC3339, startDate)
	end, _ := time.Parse(time.RFC3339, endDate)

	for it.HasNext() {
		item, _ := it.Next()
		var log AuditLog
		if err := json.Unmarshal(item.Value, &log); err != nil {
			continue
		}
		timeParsed, err := time.Parse(time.RFC3339, log.Timestamp)
		if err != nil || timeParsed.Before(start) || timeParsed.After(end) {
			continue
		}
		switch filterType {
		case "documentId":
			if log.DocumentID == filterValue {
				results = append(results, &log)
			}
		case "institution":
			if log.Institution == filterValue {
				results = append(results, &log)
			}
		case "userId":
			if log.UserID == filterValue {
				results = append(results, &log)
			}
		case "all":
			results = append(results, &log)
		default:
			continue
		}
	}
	return results, nil
}

func main() {
	cc, err := contractapi.NewChaincode(new(PublicContract))
	if err != nil {
		panic(err)
	}
	if err := cc.Start(); err != nil {
		panic(err)
	}
}