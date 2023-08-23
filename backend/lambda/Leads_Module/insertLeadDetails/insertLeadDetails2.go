package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
	"strconv"
	//SES
	"bytes"
	"net/smtp"
	"text/template"
)

const (
	host     = "ccl-psql-dev.cclxlbtddgmn.ap-south-1.rds.amazonaws.com"
	port     = 5432
	user     = "postgres"
	password = "Kasvibesc!!09"
	dbname   = "cclqadb"
	//email-SMTP
	from_email = "itsupport@continental.coffee"
	smtp_pass = "is@98765"
	// smtp server configuration.
	smtpHost = "smtp.gmail.com"
	smtpPort = "587"
)
const generalTemp=`<!DOCTYPE html>
	    <html>
		<head>
			<img src="https://s3.ap-south-1.amazonaws.com/beta-a2z.cclproducts.com/static/media/CCLEmailTemplate.png">
		<style>
		table {
  		font-family: arial, sans-serif;
  		border-collapse: collapse;
  		width: 100%;
		}
		td, th {
  		border: 1px solid #dddddd;
  		text-align: left;
  		padding: 8px;
		}
		tr:nth-child(even) {
  		background-color: #dddddd;
		}
		</style>
		</head>
		<body>
		<h3>Hi,</h3>
			<p>{{.EMessage}}</p>
		<table>
  		<tr>
    		<th>Account Name</th>
    		<th>Account Country</th>
    		<th>Account Owner</th>
  		</tr>
  		<tr>
    		<td>{{.EAccName}}</td>
    		<td>{{.EAccCountry}}</td>
    		<td>{{.EAccOwner}}</td>
  		</tr>
		</table>
		<p>Regards,</p>
		<p>a2z.cclproducts</p>
		</body>
		</html>`

type Email struct {
	
	ToEmail    string `json:"to_email"`
	ToName    string `json:"name"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
}

type LeadDetails struct {
	Update                     bool   `json:"update"`
	LeadId                     string    `json:"leadid"`
	LastIdsno					int    `json:"lastidsno"`
	Idsno						int    `json:"idsno"`
	Accountname                string `json:"accountname"`
	Aliases                    string `json:"aliases"`
	Accounttypeid              string `json:"accounttypeid"`
	Website                    string `json:"website"`
	Approximativeannualrevenue string `json:"approxannualrev"`
	Productsegmentid           string    `json:"productsegmentid"`
	ContactSalutationid        int    `json:"contact_salutationid"`
	Contactfirstname           string `json:"contact_firstname"`
	Contactlastname            string `json:"contact_lastname"`
	ContactPosition            string `json:"contact_position"`
	ContactEmail               string `json:"contact_email"`
	ContactPhone               string `json:"contact_phone"`
	ContactExtId			   string `json:"contact_ext"`
	ContactMobile              string `json:"contact_mobile"`
	Manfacunit                 int    `json:"manfacunit"`
	Instcoffee                 int    `json:"instcoffee"`
	Price                      int    `json:"sample_ready"`
	Coffeetypeid               string `json:"coffeetypeid"`
	OtherInformation           string `json:"otherinformation"`
	BillingStreetAddress       string `json:"billing_street"`
	BillingCity                string `json:"billing_citycode"`
	BillingState               string `json:"billing_statecode"`
	BillingPostalCode          string `json:"billing_postalcode"`
	BillingCountry             string `json:"billing_countrycode"`
	ContactStreetAddress       string `json:"contact_street"`
	ContactCity                string `json:"contact_citycode"`
	ContactState               string `json:"contact_statecode"`
	ContactPostalCode          string `json:"contact_postalcode"`
	ContactCountry             string `json:"contact_countrycode"`
	CreatedDate                string `json:"createddate"`
	CreatedUserid              string `json:"createduserid"`
	LdCreatedUserid            string `json:"ldcreateduserid"`
	LdCreatedUserName          string `json:"ldcreatedusername"`
	UserEmail                  string `json:"emailid"`
	ModifiedDate               string `json:"modifieddate"`
	ModifiedUserid             string `json:"modifieduserid"`
	ShippingContinentid        string `json:"shipping_continentid"`
	ShippingCountryid          string `json:"countryid"`
	Leadscore                  int    `json:"leadscore"`
	Masterstatus               string `json:"masterstatus"`
	Approvalstatus             int    `json:"approvalstatus"`
	ShippingContinent          string `json:"shipping_continent"`
	ShippingCountry            string `json:"shipping_country"`
	Isactive                   int    `json:"isactive"`
	AuditLogDetails  []AuditLogGCPO `json:"audit_log_gc_po"`
}

type LeadId struct {
	Id string `json:"leadid"`
}
type AuditLogGCPO struct {
	CreatedDate    string `json:"createddate"`
	CreatedUserid  string `json:"createduserid"`
	ModifiedDate   string `json:"modifieddate"`
	ModifiedUserid string `json:"modifieduserid"`
	Description    string `json:"description"`
}

func insertLeadDetails(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{"Access-Control-Allow-Origin": "*", "Access-Control-Allow-Headers": "Origin, X-Requested-With, Content-Type, Accept"}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var lead LeadDetails
	var email Email
	var audit AuditLogGCPO
	err := json.Unmarshal([]byte(request.Body), &lead)
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
	var rows *sql.Rows
	// Find created user username
	sqlStatementFUser1 := `SELECT username 
						FROM dbo.users_master_newpg where userid=$1`
	rows, err = db.Query(sqlStatementFUser1,lead.CreatedUserid)
	for rows.Next() {
		err = rows.Scan(&lead.LdCreatedUserName)
	}
	//Find Duplicate Lead:
	log.Println("Finding Duplicate Leadname in DB")
	var duplicatelead string
	sqlStatementDupLead := `select accountname from dbo.cms_leads_master where accountname = $1`
	rows, err = db.Query(sqlStatementDupLead,lead.Accountname)
	if err != nil {
		log.Println(err.Error())
		
	}
	for rows.Next() {
		err = rows.Scan(&duplicatelead)
	}
	
	if lead.Update {
		log.Println("Entered Update Leads Segment")
		log.Println(lead.Productsegmentid)

		sqlStatementU1 := `UPDATE dbo.cms_leads_master SET 
							accountname=$1,
							accounttypeid=$2,
							contact_mobile=$3,
							email=$4,
							phone=$5,
							modifieddate=$6,
							modifieduserid=$7,
							shipping_continentid=$8,
							countryid=$9,
							approxannualrev=$10,
							website=$11,
							productsegmentid=$12,
							leadscore=$13,
							contactfirstname=$14,
							contactlastname=$15,
							manfacunit=$16,
							instcoffee=$17,
							price=$18,
							contact_salutationid=$19,
							contact_position=$20,
							shipping_continent=$21,
							shipping_country=$22,
							coffeetypeid=$23,
							aliases=$24,
							otherinformation=$25,
							contact_ext_id=$26
							where leadid=$27`		

		_, errU1 := db.Query(sqlStatementU1, 
			lead.Accountname,
			lead.Accounttypeid,
			lead.ContactMobile,
			lead.ContactEmail,
			lead.ContactPhone,
			lead.ModifiedDate,
			lead.ModifiedUserid,
			lead.ShippingContinentid,
			lead.ShippingCountryid,
			lead.Approximativeannualrevenue,
			lead.Website,
			lead.Productsegmentid,
			lead.Leadscore,
			lead.Contactfirstname,
			lead.Contactlastname,
			lead.Manfacunit,
			lead.Instcoffee,
			lead.Price,
			lead.ContactSalutationid,
			lead.ContactPosition,
			lead.ShippingContinent,
			lead.ShippingCountry,
			lead.Coffeetypeid,
			lead.Aliases,
			lead.OtherInformation,
		    lead.ContactExtId,
			lead.LeadId)
			log.Println("Update lead successful")
		if errU1 != nil {
			log.Println(errU1.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, errU1.Error(), false}, nil
		}

		sqlStatement3 := `UPDATE dbo.cms_leads_billing_address_master SET street=$1, city=$2, stateprovince=$3, postalcode=$4, country=$5 where billingid=$6`

		rows, err = db.Query(sqlStatement3, lead.BillingStreetAddress, lead.BillingCity, lead.BillingState, lead.BillingPostalCode, lead.BillingCountry, lead.LeadId)

		sqlStatement4 := `UPDATE dbo.cms_leads_shipping_address_master SET street=$1, city=$2, stateprovince=$3, postalcode=$4, country=$5 where shippingid=$6`

		rows, err = db.Query(sqlStatement4, lead.ContactStreetAddress, lead.ContactCity, lead.ContactState, lead.ContactPostalCode, lead.ContactCountry, lead.LeadId)

		if err != nil {
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		// Insert Audit Info
		log.Println("Entered Audit Module for PO Type")
		// Find created user username
		sqlStatementAUser1 := `SELECT u.userid,u.username 
								FROM dbo.users_master_newpg u
								inner join 
								dbo.auditlog_cms_leads_master_newpg ld 
								on ld.createdby=u.userid
								where ld.leadid=$1`
		rows, err = db.Query(sqlStatementAUser1,lead.LeadId)
		for rows.Next() {
			err = rows.Scan(&lead.LdCreatedUserid,&lead.LdCreatedUserName)
		}
		audit.Description="Lead Details Modified"
		// sd.InvoiceDate = time.Now().Format("2006-01-02")
		audit.ModifiedDate=time.Now().Format("2006-01-02")
		audit.ModifiedUserid=lead.ModifiedUserid
		
		sqlStatementADT := `update
							dbo.auditlog_cms_leads_master_newpg
							set
							description=$1,
							modifiedby=$2,
							modified_date=$3
							where
							leadid=$4`
		_, errADT := db.Query(sqlStatementADT,
								audit.Description,
								audit.ModifiedUserid,
								audit.ModifiedDate,
								lead.LeadId)
				
		log.Println("Audit Insert Query Executed")
		if errADT != nil {
			log.Println("unable to insert Audit Details", errADT)
		}
		//Insert Notification
		sqlStatementNotif := `insert into dbo.notifications_master_newpg(userid,objid,feature_category,status) 
							values($1,$2,'Lead','Lead Updated')`
		_, _ = db.Query(sqlStatementNotif,lead.ModifiedUserid,lead.LeadId)

		res, _ := json.Marshal(rows)
		return events.APIGatewayProxyResponse{200, headers, nil, string(res), false}, nil
	} else if duplicatelead=="" {
		log.Println("Entered Lead creation segment")
		log.Println(lead.Productsegmentid)
		//Find latest poid
		sqlStatementLD1 := `SELECT idsno 
							FROM dbo.cms_leads_master 
							where idsno is not null
							ORDER BY idsno DESC 
							LIMIT 1`
		rows, err = db.Query(sqlStatementLD1)

		// var po InputAdditionalDetails
		for rows.Next() {
			err = rows.Scan(&lead.LastIdsno)
		}
		//Generating PO NOs----------------
		lead.Idsno = lead.LastIdsno + 1
		lead.LeadId = "Lead-" + strconv.Itoa(lead.Idsno)
		
		sqlStatement1 := `INSERT INTO dbo.cms_leads_master (
			leadid,
			autogencode,
			legacyid,
			accountname,
			accounttypeid,
			phone,
			email,
			createddate,
			createduserid,
			shipping_continentid,
			countryid,
			approxannualrev,
			website,
			productsegmentid,
			leadscore,
			masterstatus,
			contactfirstname,
			contactlastname,
			manfacunit,
			instcoffee,
			price,
			approvalstatus,
			contact_salutationid,
			contact_position,
			contact_mobile,
			shipping_continent,
			shipping_country,
			coffeetypeid,
			aliases,
			isactive,
			otherinformation,
			contact_ext_id) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28,$29,$30,$31,$32)`

		rows, err = db.Query(sqlStatement1,
			lead.LeadId,
			lead.LeadId,
			lead.Idsno,
			lead.Accountname,
			lead.Accounttypeid,
			lead.ContactPhone,
			lead.ContactEmail,
			lead.CreatedDate,
			lead.CreatedUserid,
			lead.ShippingContinentid,
			lead.ShippingCountryid,
			lead.Approximativeannualrevenue,
			lead.Website,
			lead.Productsegmentid,
			lead.Leadscore,
			lead.Masterstatus,
			lead.Contactfirstname,
			lead.Contactlastname,
			lead.Manfacunit,
			lead.Instcoffee,
			lead.Price,
			lead.Approvalstatus,
			lead.ContactSalutationid,
			lead.ContactPosition,
			lead.ContactMobile,
			lead.ShippingContinent,
			lead.ShippingCountry,
			lead.Coffeetypeid,
			lead.Aliases,
			lead.Isactive,
			lead.OtherInformation,
		    lead.ContactExtId)

		if err != nil {
			log.Println("Insert to lead table failed")
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		//Insert Notification
		sqlStatementNotif := `insert into dbo.notifications_master_newpg(userid,objid,feature_category,status) 
							values($1,$2,'Lead','Lead Created')`
		_, _ = db.Query(sqlStatementNotif,lead.CreatedUserid,lead.LeadId)


		sqlStatement3 := `INSERT INTO dbo.cms_leads_billing_address_master(
							leadid, billingid, street, city, stateprovince, postalcode, country) VALUES ($1, $2, $3, $4, $5, $6, $7)`

		rows, err = db.Query(sqlStatement3, lead.LeadId, lead.LeadId, lead.BillingStreetAddress, lead.BillingCity, lead.BillingState, lead.BillingPostalCode, lead.BillingCountry)
		log.Println("Insert into Leads Billing table")
		sqlStatement4 := `INSERT INTO dbo.cms_leads_shipping_address_master (leadid, shippingid, street, city, stateprovince, postalcode, country) VALUES ($1, $2, $3, $4, $5, $6, $7)`
		rows, err = db.Query(sqlStatement4, lead.LeadId, lead.LeadId,lead.ContactStreetAddress, lead.ContactCity, lead.ContactState, lead.ContactPostalCode, lead.ContactCountry)
		log.Println("Insert into Leads Shipping table")
		if err != nil {
			log.Println(err.Error())
			return events.APIGatewayProxyResponse{500, headers, nil, err.Error(), false}, nil
		}
		//Send email
		
		email.Subject="Lead created"  
		email.Message="New lead has been created"
		email.ToEmail= lead.UserEmail
		smtpSendEmail(generalTemp,email.Subject,email.Message,lead.Accountname,lead.ShippingCountry,lead.LdCreatedUserName,email.ToEmail)
		//------------------Insert Audit Info----------------------------
		log.Println("Entered Audit Module for Lead Module")
		audit.CreatedUserid = lead.CreatedUserid
		audit.CreatedDate = lead.CreatedDate
		audit.Description = "Lead Created"
		sqlStatementADT := `INSERT INTO dbo.auditlog_cms_leads_master_newpg(
						leadid,createdby, created_date, description)
						VALUES($1,$2,$3,$4)`
		_, errADT := db.Query(sqlStatementADT,
			lead.LeadId,
			audit.CreatedUserid,
			audit.CreatedDate,
			audit.Description)

		log.Println("Audit Insert Query Executed")
		if errADT != nil {
			log.Println("unable to insert Audit Details", errADT)
		}

		// res, _ := json.Marshal(value)
		return events.APIGatewayProxyResponse{200, headers, nil, string("Lead saved successfully"), false}, nil

		
	} else {
		return events.APIGatewayProxyResponse{230, headers, nil, "Lead Name already exists", false}, nil
	}
	return events.APIGatewayProxyResponse{200, headers, nil, string("Success"), false}, nil
}
func main() {
	lambda.Start(insertLeadDetails)
}
func smtpSendEmail(temp,subject,message,accname,acccountry,accowner,to_email string) (string, error) {
	log.Println("Entered SMTP Email Module")
  // Receiver email address.
  to := []string{
    to_email,
  }
  // Authentication.
  auth := smtp.PlainAuth("", from_email, smtp_pass, smtpHost)

  t := template.Must(template.New(temp).Parse(temp))
 
  var body bytes.Buffer

  mimeHeaders := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
  body.Write([]byte(fmt.Sprintf("Subject:"+subject+"\n%s\n\n",mimeHeaders)))
//   body.Write([]byte(fmt.Sprintf("Subject: This is a test subject \n%s\n\n", mimeHeaders)))

  t.Execute(&body, struct {
   
    EMessage string
	EAccName string
	EAccCountry string
    EAccOwner string
	
  }{
    
    EMessage: message,
	EAccName:    accname,
	EAccCountry: acccountry,
	EAccOwner: accowner,
  })

  // Sending email.
  err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from_email, to, body.Bytes())
  if err != nil {
    fmt.Println(err)
    
  }
  return "Email Sent!",nil
}
