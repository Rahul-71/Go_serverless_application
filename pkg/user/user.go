package user

import (
	"encoding/json"
	"errors"

	"github.com/Rahul-71/go-serverless/pkg/validators"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

var (
	ErrorFailedToFetchRecord     = "failed to fetch record"
	ErrorFailedToUnmarshalRecord = "failed to unmarshal record"
	ErrorInvalidUserData         = "invalid user data"
	ErrorInvalidEmail            = "invalid email"
	ErrorMarshalItem             = "could not marshal item"
	ErrorDeleteItem              = "could not delete item"
	ErrorDynamoPutItem           = "could not dynamo put item"
	ErrorUserAlreadyExists       = "user already exists"
	ErrorUserDoesNotExists       = "user does not exists"
)

type User struct {
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func FetchUser(email, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*User, error) {

	// based on some key we'll run operation in db. In this case, user will be found in db based
	// on its mailId
	input := dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"email": {S: aws.String(email)},
		},
		TableName: aws.String(tableName),
	}

	result, err := dynaClient.GetItem(&input)
	if err != nil {
		return nil, errors.New(ErrorFailedToFetchRecord)
	}

	item := new(User)
	err = dynamodbattribute.UnmarshalMap(result.Item, item) // taking User from result & marshalling into item of type User
	if err != nil {
		return nil, errors.New(ErrorFailedToUnmarshalRecord)
	}

	return item, nil

}

func FetchUsers(tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*[]User, error) {
	input := dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}
	result, err := dynaClient.Scan(&input)
	if err != nil {
		return nil, errors.New(ErrorFailedToFetchRecord)
	}

	item := new([]User)
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, item)
	if err != nil {
		return nil, errors.New(ErrorFailedToUnmarshalRecord)
	}

	return item, nil
}

func CreateUser(req events.APIGatewayProxyRequest, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*User, error) {
	var createuser User

	if err := json.Unmarshal([]byte(req.Body), &createuser); err != nil {
		return nil, errors.New(ErrorInvalidUserData)
	}
	// check users email is valid or not
	if !validators.IsEmailValid(createuser.Email) {
		return nil, errors.New(ErrorInvalidEmail)
	}

	// check if user already exists
	curruser, _ := FetchUser(createuser.Email, tableName, dynaClient)
	if curruser != nil && len(curruser.Email) != 0 {
		return nil, errors.New(ErrorUserAlreadyExists)
	}

	// if everything is OK, let's marhsal the request into data that dynamodb can understand
	attrVal, err := dynamodbattribute.MarshalMap(createuser)
	if err != nil {
		return nil, errors.New(ErrorMarshalItem)
	}

	// let's create the input for dynamodb
	input := dynamodb.PutItemInput{
		Item:      attrVal,
		TableName: aws.String(tableName),
	}

	// dynaClient will trigger the operation to run PUT item to dynamodb
	_, err = dynaClient.PutItem(&input)
	if err != nil {
		return nil, errors.New(ErrorDynamoPutItem)
	}

	return &createuser, nil
}

func UpdateUser(req events.APIGatewayProxyRequest, tableName string, dynaClient dynamodbiface.DynamoDBAPI) (*User, error) {

	var updateuser User

	if err := json.Unmarshal([]byte(req.Body), &updateuser); err != nil {
		return nil, errors.New(ErrorInvalidUserData)
	}

	// first check if user exist & with correct data
	curruser, _ := FetchUser(updateuser.Email, tableName, dynaClient)
	if curruser != nil && len(curruser.Email) == 0 {
		return nil, errors.New(ErrorUserDoesNotExists)
	}

	// convert unmarshalled data from json to data that dynamodb can understand
	attrbVal, err := dynamodbattribute.MarshalMap(updateuser)
	if err != nil {
		return nil, errors.New(ErrorMarshalItem)
	}

	// create input that can go inside dynamodb
	input := dynamodb.PutItemInput{
		Item:      attrbVal,
		TableName: aws.String(tableName),
	}

	// use dynaClient to trigger dynamodb function to put item
	_, err = dynaClient.PutItem(&input)
	if err != nil {
		return nil, errors.New(ErrorDynamoPutItem)
	}

	return &updateuser, nil

}

func DeleteUser(req events.APIGatewayProxyRequest, tableName string, dynaClient dynamodbiface.DynamoDBAPI) error {

	email := req.QueryStringParameters["email"]
	input := dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"email": {S: aws.String(email)},
		},
		TableName: new(string),
	}

	_, err := dynaClient.DeleteItem(&input)
	if err != nil {
		return errors.New(ErrorDeleteItem)
	}
	return nil
}
