//Vehicle no added to insert query-Aug27
//New fields added to MRIN form-Sep5
//Updated with APStatus & QCStatus - Sep7
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	// "github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Kasvibesc!!09"
	dbname   = "ccldevdb"
)

type MrinDetails struct {
	Type                string `json:"type"`
	ItemID              string `json:"itemid"`
	Role                string `json:"role"`
	CreateMRIN          bool   `json:"createmrin"`
	Update              bool   `json:"update"`
	PoId                string `json:"poid"`
	PoNO                string `json:"pono"`
	Detid               string `json:"detid"`
	RelatedDetid        string `json:"related_detid"`
	ExpectedQuantity    string `json:"expected_quantity"`
	InvoiceQuantity     string `json:"invoice_quantity"`
	InvoiceNo           string `json:"invoice_no"`
	InvoiceDate         string `json:"invoice_date"`
	DeliveryDate        string `json:"delivery_date"`
	DeliveredQuantity   string `json:"delivered_quantity"`
	WaymentShortage     string `json:"wayment_shortage"`
	BalanceQuantity     string `json:"balance_quantity"`
	TotalBalQuant       string `json:"total_balance_quantity"`
	CreatedDate         string `json:"createdon"`
	CreatedBy           string `json:"createdby"`
	Status              string `json:"status"`
	QCStatus            string `json:"qcStatus"`
	RequestApproval     string `json:"reqapproval"`
	VendorId            string `json:"vendor_id"`
	EntityId            string `json:"entityid"`
	LastMrinidsno       int    `json:"last_mrinidsno"`
	Mrinid              string `json:"mrinid"`
	Mrinidsno           int    `json:"mrinidsno"`
	LastVgidsno         int    `json:"last_vgidsno"`
	Vgidsno             int    `json:"vgidsno"`
	Vgcompid            string `json:"vgcompid"`
	Mrinno              string `json:"mrinno"`
	Mrindate            string `json:"mrindate"`
	LastDetId           int    `json:"last_detid"`
	VehicleNo           string `json:"vehicle_no"`
	WaymentShortageFlag bool   `json:"wayment_shortage_flag"`
	//New Form changes -Sep5
	APDetails     string `json:"apDetails"`
	APStatus      string `json:"apStatus"`
	WayBillNo     string `json:"wayBillNumber"`
	WayBillDate   string `json:"wayBillDate"`
	InvoiceAmount string `json:"invoiceAmount"`
	Locations     string `json:"location"`

	//Doc upload
	FileName     string `json:"file_name"`
	DocKind      string `json:"doc_kind"`
	DocumentName string `json:"document_name"`
	FileContent  string `json:"document_content"`
	// DeliveredSpec []DeliveredGCSpec`json:"delivered_spec"`
	Del_Density       string `json:"del_density"`
	Del_Moisture      string `json:"del_moisture"`
	Del_Browns        string `json:"del_browns"`
	Del_Blacks        string `json:"del_blacks"`
	Del_BrokenBits    string `json:"del_brokenbits"`
	Del_InsectedBeans string `json:"del_insectedbeans"`
	Del_Bleached      string `json:"del_bleached"`
	Del_Husk          string `json:"del_husk"`
	Del_Sticks        string `json:"del_sticks"`
	Del_Stones        string `json:"del_stones"`
	Del_BeansRetained string `json:"del_beansretained"`
}

// type DeliveredGCSpec struct{

// }
type MRINStatus struct {
	TotalMRINs  int    `json:"mrins_count"`
	StatusCount int    `json:"mrins_count"`
	TotalAP     string `json:"total_amount_payable"`
}
type QuantityCalculation struct {
	ExpectedQuantity string `json:"expected_quantity"`
	ReceivedQuantity string `json:"received_quantity"`
	IsMatched        bool   `json:"is_matched"`
}
type MRINDocumentDetails struct {
	DocumentName string `json:"document_name"`
	FileName     string `json:"file_name"`
	DocKind      string `json:"doc_kind"`
}

type LastDocDetails struct {
	DocIdno int `json:"docid_no"`
}

type FileResponse struct {
	FileName        string `json:"fileName"`
	FileLink        string `json:"fileLink"`
	FileData        string `json:"fileData"`
	FileContentType string `json:"fileContentType"`
}

func NewNullString(s string) sql.NullString {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}
func NewNullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{
		Time:  t,
		Valid: true,
	}
}

func insertOrUpdateMrin(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var sd MrinDetails
	// var dgc DeliveredGCSpec
	err := json.Unmarshal([]byte(request.Body), &sd)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}
	defer db.Close()

	// check db
	err = db.Ping()

	if err != nil {
		log.Println(err)
		return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
	}

	fmt.Println("Connected!")
	// var rows *sql.Rows
	var documentDetail MRINDocumentDetails
	if sd.Mrinid != "" {
		sqlStatementFPO := `select pono from dbo.inv_gc_po_mrin_master_newpg 
							where mrinid=$1`
		rowsFPO, errFPO := db.Query(sqlStatementFPO, sd.Mrinid)
		if errFPO != nil {
			log.Println("Unable to get PO for MRIN", errFPO.Error())
		}
		defer rowsFPO.Close()
		for rowsFPO.Next() {
			errFPO = rowsFPO.Scan(&sd.PoNO)
		}
	}
	if sd.Update {
		log.Println("Update Module Entered")
		sqlStatement1 := `UPDATE dbo.inv_gc_po_mrin_master_newpg 
						  SET 
						  vehicleno=$1,
						  invoiceno=$2,
						  invoice_quantity=$3,
						  invoicedate=$4,
						  delivery_date=$5,
						  delivered_quantity=$6,
						  wayment_shortage=$7,
						  balance_quantity=$8,
						  waybillno=$9,
						  waybilldate=$10,
						  location=$11
						  where 
						  mrinid=$12`

		_, err = db.Query(sqlStatement1,
			sd.VehicleNo,
			NewNullString(sd.InvoiceNo),
			NewNullString(sd.InvoiceQuantity),
			sd.InvoiceDate,
			sd.DeliveryDate,
			NewNullString(sd.DeliveredQuantity),
			NewNullString(sd.WaymentShortage),
			NewNullString(sd.BalanceQuantity),
			sd.WayBillNo,
			sd.WayBillDate,
			NewNullString(sd.Locations),
			sd.Mrinid)

		if err != nil {
			log.Println("unable to update mrin", err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		//------MRIN Doc Upload Module-----------------------
		log.Println("MRIN Upload Doc section entered")
		sqlStatement2 := `delete from dbo.pur_gc_po_master_documents where mrinid=$1`
		_, err := db.Query(sqlStatement2, sd.Mrinid)
		// if err2 != nil {
		// 	log.Println("Unable to Delete the documents for MRIN")
		// 	return events.APIGatewayProxyResponse{500, headers, nil, err2.Error(), false}, nil
		// }
		log.Println("Successfully removed file in db with for MRIN: ", sd.Mrinid)
		// return events.APIGatewayProxyResponse{200, headers, nil, string("Removed Successfully"), false}, nil

		sqlStatementD1 := `select docidsno from dbo.pur_gc_mrin_master_documents_newpg
							where docidsno is not null
							order by docidsno DESC LIMIT 1`
		rowsD1, errD1 := db.Query(sqlStatementD1)

		if errD1 != nil {
			log.Println("Unable to get last updated id")
			return events.APIGatewayProxyResponse{500, headers, nil, errD1.Error(), false}, nil
		}

		var lastDoc LastDocDetails
		for rowsD1.Next() {
			errD1 = rowsD1.Scan(&lastDoc.DocIdno)
		}

		docIdsno := lastDoc.DocIdno + 1
		docId := "MRIN_INV_DOC-" + strconv.Itoa(docIdsno)
		fileName := "MRIN_Document_GC_" + docId + ".pdf"

		sqlStatement3 := `INSERT INTO dbo.pur_gc_mrin_master_documents_newpg (docid, docidsno, pono,detid, docname, filename, dockind,mrinid) VALUES ($1, $2, $3, $4, $5, $6,$7,$8)`
		_, err = db.Query(sqlStatement3, docId, docIdsno, sd.PoNO, sd.Detid, sd.DocumentName, fileName, sd.DocKind, sd.Mrinid)

		if err != nil {
			log.Println("Insert to po document master failed")
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		log.Println("Successfully uploaded file in db with ", docIdsno, docId, fileName)

		k, _ := uploadDocToS3(sd.FileContent, fileName)
		log.Println("Successfully uploaded file in s3 bucket ", k, fileName)

		//Insert Notification
		sqlStatementNotif1 := `insert into dbo.notifications_master_newpg(userid,objid,feature_category,status) 
							values($1,$2,'MRIN','MRIN Updated')`
		_, _ = db.Query(sqlStatementNotif1, sd.CreatedBy, sd.Mrinid)
		return events.APIGatewayProxyResponse{200, headers, nil, string(fileName), false}, nil

		// res, _ := json.Marshal(rows)
		// return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil

		// else if (sd.Role=="QC Manager" || sd.Role=="Managing Director") {
		// log.Println("Entered QC Manager Update Code")

	} else if (sd.Type == "Approve") && (sd.Role == "QC Manager" || sd.Role == "Managing Director") {
		log.Println("MRIN QC Approval Module Entered")

		sqlStatementQ1 := `UPDATE dbo.inv_gc_po_mrin_master_newpg 
						  SET 
						  status=$1,
						  qc_status=$2
						  where 
						  mrinid=$3`

		_, errQ1 := db.Query(sqlStatementQ1,
			"Pending with Finance",
			sd.QCStatus,
			sd.Mrinid)

		if errQ1 != nil {
			log.Println("unable to update mrin by QC Dept User", errQ1.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errQ1.Error(), false}, nil
		}
		//----------Updating PO balance quantity when MRIN is approved----------
		sqlStatementQ2 := `update dbo.pur_gc_po_con_master_newpg
							set
							balance_quantity=balance_quantity-(select delivered_quantity from dbo.inv_gc_po_mrin_master_newpg where mrinid=$1)
							where
							pono=$2`
		_, errQ2 := db.Query(sqlStatementQ2,
			sd.Mrinid, sd.PoNO)

		if errQ2 != nil {
			log.Println("unable to update PO Total balance quantity from MRIN delivered quantity", errQ2.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errQ2.Error(), false}, nil
		}

		// res, _ := json.Marshal(rowsQ1)
		return events.APIGatewayProxyResponse{200, headers, nil, string("Update successful"), false}, nil

	} else if (sd.Type == "Paid") && (sd.Role == "Accounts Manager" || sd.Role == "Accounts Executive" || sd.Role == "Managing Director") {
		log.Println("MRIN Finance Approval Module Entered")

		sqlStatementF1 := `UPDATE dbo.inv_gc_po_mrin_master_newpg 
						  SET 
						  ap_details=$1,
						  ap_status=$2,
						  invoice_amount=$3,
						  status=$4
						  where 
						  mrinid=$5`

		_, errF1 := db.Query(sqlStatementF1,
			NewNullString(sd.APDetails),
			"Paid",
			NewNullString(sd.InvoiceAmount),
			"Closed",
			sd.Mrinid)

		if errF1 != nil {
			log.Println("unable to update mrin by Finance User", errF1.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errF1.Error(), false}, nil
		}
		log.Println("Checking MRINs Consolidation Status")
		var ms MRINStatus

		sqlStatementMC := `select count(detid) from dbo.pur_gc_po_dispatch_master_newpg
							where pono=$1`

		rowsMC, errMC := db.Query(sqlStatementMC, sd.PoNO)
		if errMC != nil {
			log.Println("Unable to get MRINs Count", errMC.Error())
		}
		defer rowsMC.Close()
		for rowsMC.Next() {
			errMC = rowsMC.Scan(&ms.TotalMRINs)
		}
		log.Println("Total MRINs created count: ", ms.TotalMRINs)
		sqlStatementSC := `select count(mrinid),sum(invoice_amount) from dbo.inv_gc_po_mrin_master_newpg 
								where (status like 'Closed') and pono=$1`
		rowsSC, errSC := db.Query(sqlStatementSC, sd.PoNO)
		if errSC != nil {
			log.Println("Unable to get Status Count", errSC.Error())
		}
		defer rowsSC.Close()
		for rowsSC.Next() {
			errSC = rowsSC.Scan(&ms.StatusCount, &ms.TotalAP)
		}
		log.Println("Total MRINs with status Closed count: ", ms.StatusCount)
		if ms.TotalMRINs == ms.StatusCount {
			log.Println("Total MRINs matched with Total MRIns with closed status")
			log.Println("Updating PO Consolidated Status to Paid & Approved")
			sqlStatement9 := `update dbo.pur_gc_po_con_master_newpg
								 set 
								accpay_status='Paid',
								qc_status ='Approved',
								payable_amount=$1
								where pono=$2`
			_, err9 := db.Query(sqlStatement9, ms.TotalAP, sd.PoNO)

			if err9 != nil {
				log.Println("Unable to update status", err9.Error())
			}

		}

		// res, _ := json.Marshal(rowsF1)
		return events.APIGatewayProxyResponse{200, headers, nil, string("Update successful"), false}, nil

		// } else
	} else if sd.Type == "removeDocument" {

		sqlStatement := `delete from dbo.pur_gc_po_master_documents where filename=$1`
		_, err = db.Query(sqlStatement, sd.FileName)

		log.Println("Successfully removed file in db with ", documentDetail.FileName)
		return events.APIGatewayProxyResponse{200, headers, nil, string("Removed Successfully"), false}, nil
	} else if sd.CreateMRIN {

		log.Println("Create MRIN Module Entered")

		sqlStatementDT1 := `select mrinidsno from dbo.inv_gc_po_mrin_master_newpg 
							where mrinidsno is not null 
							order by mrinidsno desc limit 1`

		rows1, err1 := db.Query(sqlStatementDT1)
		if err1 != nil {
			log.Println("Finding Next IDSNO in MRIN table failed", err1.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err1.Error(), false}, nil
		}

		// var po InputAdditionalDetails
		for rows1.Next() {
			err = rows1.Scan(&sd.LastMrinidsno)
		}

		sd.Mrinidsno = sd.LastMrinidsno + 1
		sd.Mrinid = "MRINID-" + strconv.Itoa(sd.Mrinidsno)
		sd.Mrinno = strconv.Itoa(sd.Mrinidsno) + "/" + strconv.Itoa(time.Now().Year()) + "-" + strconv.Itoa(time.Now().Year()+1)
		// sd.InvoiceDate = time.Now().Format("2006-01-02")
		sd.Status = "Pending with QC Approval"
		log.Println("Mrinidsno", sd.Mrinidsno, "Mrinid", sd.Mrinid, "Mrinno", sd.Mrinno, "Mrindate", sd.Mrindate)
		log.Println("APDETAILS:", sd.APDetails, "APSTATUS:", sd.APStatus, "WAYBILLNO:", sd.WayBillNo, "WAYBILLDATE", sd.WayBillDate, "INVOICEAMNT:", sd.InvoiceAmount)

		sqlStatement2 := `INSERT INTO dbo.inv_gc_po_mrin_master_newpg (
			poid,
			pono,
			detid,
			vehicleno,
			related_detid,
			expected_quantity,
			invoiceno,
			invoice_quantity,
			delivery_date,
			delivered_quantity,
			wayment_shortage,
			balance_quantity,
			createdon,
			createdby,
			status,
			mrinid,
			mrinidsno,
			mrinno,
			mrindate,
			invoicedate,
			entityid,
			reqapproval,
			approvalstatus, 
			vendorid,
			ap_details,
			ap_status,
			qc_status,
			waybillno,
			waybilldate,
			invoice_amount,
			location
			) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,$15,$16,$17,$18, $19, $20, $21, $22,$23,$24,$25,$26,$27,$28,$29,$30,$31)`

		rows2, err := db.Query(sqlStatement2,
			sd.PoId,
			sd.PoNO,
			sd.Detid,
			sd.VehicleNo,
			sd.RelatedDetid,
			NewNullString(sd.ExpectedQuantity),
			sd.InvoiceNo,
			NewNullString(sd.InvoiceQuantity),
			sd.DeliveryDate,
			sd.DeliveredQuantity,
			sd.WaymentShortage,
			NewNullString(sd.BalanceQuantity),
			sd.CreatedDate,
			sd.CreatedBy,
			sd.Status,
			sd.Mrinid,
			sd.Mrinidsno,
			sd.Mrinno,
			sd.Mrindate,
			sd.InvoiceDate,
			sd.EntityId,
			false,
			false,
			sd.VendorId,
			NewNullString(sd.APDetails),
			sd.APStatus,
			sd.QCStatus,
			sd.WayBillNo,
			sd.WayBillDate,
			NewNullString(sd.InvoiceAmount),
			sd.Locations)
		defer rows2.Close()

		if err != nil {
			log.Println("Insert to MRIN table failed", err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		if (sd.Type == "uploadDocument") && (sd.Detid != "") {
			//------MRIN Doc Upload Module-----------------------
			log.Println("MRIN Upload Doc section entered")
			if sd.Update {
				sqlStatement := `delete from dbo.pur_gc_po_master_documents where mrinid=$1`
				_, err = db.Query(sqlStatement, sd.Mrinid)

				log.Println("Successfully removed file in db with for MRIN: ", sd.Mrinid)
				return events.APIGatewayProxyResponse{200, headers, nil, string("Removed Successfully"), false}, nil
			}
			sqlStatement := `select docidsno from dbo.pur_gc_mrin_master_documents_newpg
								where docidsno is not null
								order by docidsno DESC LIMIT 1`
			rows1, err1 := db.Query(sqlStatement)

			if err1 != nil {
				log.Println("Unable to get last updated id")
				return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
			}

			var lastDoc LastDocDetails
			for rows1.Next() {
				err = rows1.Scan(&lastDoc.DocIdno)
			}

			docIdsno := lastDoc.DocIdno + 1
			docId := "MRIN_INV_DOC-" + strconv.Itoa(docIdsno)
			fileName := "MRIN_Document_GC_" + docId + ".pdf"

			sqlStatement1 := `INSERT INTO dbo.pur_gc_mrin_master_documents_newpg (docid, docidsno, pono,detid, docname, filename, dockind,mrinid) VALUES ($1, $2, $3, $4, $5, $6,$7,$8)`
			_, err = db.Query(sqlStatement1, docId, docIdsno, sd.PoNO, sd.Detid, sd.DocumentName, fileName, sd.DocKind, sd.Mrinid)

			if err != nil {
				log.Println("Insert to po document master failed")
				return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
			}
			log.Println("Successfully uploaded file in db with ", docIdsno, docId, fileName)

			k, _ := uploadDocToS3(sd.FileContent, fileName)
			log.Println("Successfully uploaded file in s3 bucket ", k, fileName)
			return events.APIGatewayProxyResponse{200, headers, nil, string(fileName), false}, nil
		}

		//-----------END OF DOC UPLOAD MODULE----------------//
		//-------AUTOMATIC DISPATCH CREATION MODULE---------//
		remainingBalance, _ := strconv.ParseFloat(sd.BalanceQuantity, 4)
		waymentBalance, _ := strconv.ParseFloat(sd.WaymentShortage, 4)
		log.Println("RemainingBalance:", remainingBalance, "WaymentBalance:", waymentBalance)
		if remainingBalance > 0 || waymentBalance > 0 {
			log.Println("Found remaining balance is not ZERO.")
			log.Println("started creating new dispatch detail")
			//Find latest DETID
			sqlStatementDT1 := `select detidsno from dbo.pur_gc_po_dispatch_master_newpg 
								order by detidsno desc limit 1`
			rows3, _ := db.Query(sqlStatementDT1)

			for rows3.Next() {
				err = rows3.Scan(&sd.LastDetId)
			}

			log.Println("Last DETIDSNO from table:", sd.LastDetId)
			newDetIdsNo := sd.LastDetId + 1
			log.Println("New DETIDSNO from table:", newDetIdsNo)
			newDetID := "GCDIS-" + strconv.Itoa(newDetIdsNo)

			var quantity string

			if sd.WaymentShortageFlag {
				log.Println("FOUND WAYMENT SHORTAGE BUT OK TO LOSE")
				quantity = sd.BalanceQuantity
			} else {
				log.Println("FOUND WAYMENT SHORTAGE BUT NOT OK.SO ADDING TO BALANCE")
				K := waymentBalance + remainingBalance
				quantity = fmt.Sprintf("%f", K)
			}

			log.Println("Quantity:", quantity)
			sqlStatement5 := `insert into dbo.pur_gc_po_dispatch_master_newpg(
			pono,
			detid,
			detidsno,
			quantity,
			dispatch_count,
			dispatch_type, parent_detid) values($1,$2,$3,$4,$5,$6,$7)`
			_, err = db.Query(sqlStatement5,
				sd.PoNO,
				newDetID,
				newDetIdsNo,
				quantity,
				"1",
				"Single", sd.Detid)
			log.Println("New DETID from table:", newDetID)
		}

		var quantity QuantityCalculation

		sqlStatement6 := `select (SUM(delivered_quantity) + SUM(wayment_shortage)) as total_quantity 
		from dbo.inv_gc_po_mrin_master_newpg where pono=$1`
		rows4, _ := db.Query(sqlStatement6, sd.PoNO)

		for rows4.Next() {
			err = rows4.Scan(&quantity.ReceivedQuantity)
		}

		log.Println("ReceivedQuantity:", quantity.ReceivedQuantity)

		sqlStatement7 := `select total_quantity from dbo.pur_gc_po_con_master_newpg where pono=$1`
		rows5, _ := db.Query(sqlStatement7, sd.PoNO)

		for rows5.Next() {
			err = rows5.Scan(&quantity.ExpectedQuantity)
		}

		log.Println("ExpectedQuantity:", quantity.ExpectedQuantity)

		if quantity.ReceivedQuantity == quantity.ExpectedQuantity {

			log.Println("Updating to shipped status:", quantity.ReceivedQuantity, quantity.ExpectedQuantity)
			sqlStatement8 := `update dbo.pur_gc_po_con_master_newpg set status ='4' where pono=$1`
			_, err := db.Query(sqlStatement8, sd.PoNO)

			if err != nil {
				log.Println("Unable to update status", err.Error())
			}

		}

		return events.APIGatewayProxyResponse{200, headers, nil, string("success"), false}, nil
	}
	return events.APIGatewayProxyResponse{200, headers, nil, string("success"), false}, nil
}
func uploadDocToS3(data string, fileDir string) (string, error) {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("ap-south-1"),
	})

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)
	dec, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Println(err)
		return "", err
	}

	s3Output, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("ccl-lambda-bucket"),
		Key:    aws.String("ccl-lambda-bucket" + "/" + fileDir),
		Body:   bytes.NewReader(dec),
	})
	if err != nil {
		log.Println(err)
		return "", err
	}
	log.Println(s3Output)
	log.Println("fileLocation: " + s3Output.Location)
	return s3Output.Location, nil
}

func main() {
	lambda.Start(insertOrUpdateMrin)
}
