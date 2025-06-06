package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type PrivateContract struct {
	contractapi.Contract
}

type PrivateDocument struct {
	DocumentID   string `json:"documentId"`
	Institution  string `json:"institution"`
	UserID       string `json:"userId"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	Hash         string `json:"hash"`
	State        string `json:"state"`
}

type PrivateAuditLog struct {
	TxID        string `json:"txID"`
	DocumentID  string `json:"documentId"`
	Institution string `json:"institution"`
	UserID      string `json:"userId"`
	Operation   string `json:"operation"`
	OldState    string `json:"oldState,omitempty"`
	NewState    string `json:"newState"`
	Timestamp   string `json:"timestamp"`
}

func (c *PrivateContract) SavePrivateDocument(ctx contractapi.TransactionContextInterface, documentId, institution, userId, name, path, hash, state string) error {
	doc := PrivateDocument{DocumentID: documentId, Institution: institution, UserID: userId, Name: name, Path: path, Hash: hash, State: state}
	bytes, _ := json.Marshal(doc)
	if err := ctx.GetStub().PutPrivateData("collectionPrivateDocs", documentId, bytes); err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()
	ts, _ := ctx.GetStub().GetTxTimestamp()
	log := PrivateAuditLog{
		TxID:        txID,
		DocumentID:  documentId,
		Institution: institution,
		UserID:      userId,
		Operation:   "create",
		NewState:    state,
		Timestamp:   time.Unix(ts.Seconds, int64(ts.Nanos)).Format(time.RFC3339),
	}
	logBytes, _ := json.Marshal(log)
	return ctx.GetStub().PutPrivateData("collectionAuditLogs", "AUDIT_"+txID, logBytes)
}

func (c *PrivateContract) UpdateDocumentState(ctx contractapi.TransactionContextInterface, documentId, newState string) error {
	data, err := ctx.GetStub().GetPrivateData("collectionPrivateDocs", documentId)
	if err != nil || data == nil {
		return fmt.Errorf("document not found")
	}
	var doc PrivateDocument
	_ = json.Unmarshal(data, &doc)
	oldState := doc.State
	doc.State = newState
	updated, _ := json.Marshal(doc)
	if err := ctx.GetStub().PutPrivateData("collectionPrivateDocs", documentId, updated); err != nil {
		return err
	}
	txID := ctx.GetStub().GetTxID()
	ts, _ := ctx.GetStub().GetTxTimestamp()
	log := PrivateAuditLog{
		TxID:        txID,
		DocumentID:  documentId,
		Institution: doc.Institution,
		UserID:      doc.UserID,
		Operation:   "update_state",
		OldState:    oldState,
		NewState:    newState,
		Timestamp:   time.Unix(ts.Seconds, int64(ts.Nanos)).Format(time.RFC3339),
	}
	logBytes, _ := json.Marshal(log)
	return ctx.GetStub().PutPrivateData("collectionAuditLogs", "AUDIT_"+txID, logBytes)
}

func (c *PrivateContract) GetPrivateDocumentByID(ctx contractapi.TransactionContextInterface, documentId string) (*PrivateDocument, error) {
	data, err := ctx.GetStub().GetPrivateData("collectionPrivateDocs", documentId)
	if err != nil || data == nil {
		return nil, fmt.Errorf("not found")
	}
	var doc PrivateDocument
	_ = json.Unmarshal(data, &doc)
	return &doc, nil
}

func (c *PrivateContract) QueryPrivateByInstitution(ctx contractapi.TransactionContextInterface, institution string) ([]*PrivateDocument, error) {
	iterator, _ := ctx.GetStub().GetPrivateDataByRange("collectionPrivateDocs", "", "")
	var results []*PrivateDocument
	for iterator.HasNext() {
		record, _ := iterator.Next()
		var doc PrivateDocument
		if json.Unmarshal(record.Value, &doc) == nil && doc.Institution == institution {
			results = append(results, &doc)
		}
	}
	return results, nil
}

func (c *PrivateContract) QueryPrivateByUser(ctx contractapi.TransactionContextInterface, userId string) ([]*PrivateDocument, error) {
	iterator, _ := ctx.GetStub().GetPrivateDataByRange("collectionPrivateDocs", "", "")
	var results []*PrivateDocument
	for iterator.HasNext() {
		record, _ := iterator.Next()
		var doc PrivateDocument
		if json.Unmarshal(record.Value, &doc) == nil && doc.UserID == userId {
			results = append(results, &doc)
		}
	}
	return results, nil
}

func (c *PrivateContract) QueryAuditLogs(ctx contractapi.TransactionContextInterface, filterType, filterValue, startDate, endDate string) ([]*PrivateAuditLog, error) {
	iterator, _ := ctx.GetStub().GetPrivateDataByRange("collectionAuditLogs", "AUDIT_", "AUDIT_z")
	var logs []*PrivateAuditLog
	start, _ := time.Parse(time.RFC3339, startDate)
	end, _ := time.Parse(time.RFC3339, endDate)

	for iterator.HasNext() {
		record, _ := iterator.Next()
		var log PrivateAuditLog
		if err := json.Unmarshal(record.Value, &log); err != nil {
			continue
		}
		timeParsed, err := time.Parse(time.RFC3339, log.Timestamp)
		if err != nil || timeParsed.Before(start) || timeParsed.After(end) {
			continue
		}
		switch filterType {
		case "documentId":
			if log.DocumentID == filterValue {
				logs = append(logs, &log)
			}
		case "institution":
			if log.Institution == filterValue {
				logs = append(logs, &log)
			}
		case "userId":
			if log.UserID == filterValue {
				logs = append(logs, &log)
			}
		case "all":
			logs = append(logs, &log)
		default:
			continue
		}
	}
	return logs, nil
}

func main() {
	cc, err := contractapi.NewChaincode(new(PrivateContract))
	if err != nil {
		panic(err)
	}
	if err := cc.Start(); err != nil {
		panic(err)
	}
}
