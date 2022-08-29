package main
import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)
 // Define structs to be used by chaincode
 type Student struct {
	 StudentId  string `json:"studentId"`
	 Owner       string `json:"owner"`
	 StudentURI  string `json:"studentURI"`
 }
 
 type Transferlog struct {
	 From        string `json:"from"`
	 To          string `json:"to"`
	 StudentId string `json:"studentId"`
 }

 const totalPrefix = "totalstudent"
 const studentPrefix = "student"

 const database = "Studentcollection1"

 type SmartContract struct {
	contractapi.Contract
 }


func (s *SmartContract) Initialize(ctx contractapi.TransactionContextInterface, database1 string) (bool, error) {
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return false, fmt.Errorf("failed to get an clientmspid. %v ",err)
	}
	if clientMSPID != "Org1MSP" {
		return false, fmt.Errorf("Client is not authorized for the database.")
	}

	bytes, err := ctx.GetStub().GetState(database)
	if err != nil {
		return false, fmt.Errorf("failed to get database name. %v ",err)
	}

	if bytes != nil {
		return false, fmt.Errorf("Contract name already set.")
	}

	err = ctx.GetStub().PutState(database , []byte(database1))
	if err != nil {
		return false, fmt.Errorf("failed to PutState databasename %s: %v", database, err)
	}

	return true, nil

}

func (s* SmartContract) MintStudent(ctx contractapi.TransactionContextInterface, studentURI string, studentId string) (*Student, error) {
	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to get an clientmspid. %v ",err)
	}
	if clientMSPID != "Org1MSP" {
		return nil, fmt.Errorf("Client is not authorized for the database.")
	}
	minter64, err := ctx.GetClientIdentity().GetID()
	minterBytes, err := base64.StdEncoding.DecodeString(minter64)
	if err != nil {
		return nil, fmt.Errorf("failed to DecodeString minter. : %v", err)
	}
	minter := string(minterBytes)
	
	a := _studentExists(ctx, studentId)
	if a {
		return nil, fmt.Errorf("the student %s is already minted.: %v", studentId, err)
	}

	// Add a non-fungible token
	student := new(Student)
	student.StudentId = studentId
	student.Owner = minter
	student.StudentURI = studentURI

	studentKey, err := ctx.GetStub().CreateCompositeKey(studentPrefix, []string{studentId})
	if err != nil {
		return nil, fmt.Errorf("failed to CreateCompositeKey to studentKey.: %v", err)
	}

	studentBytes, err := json.Marshal(student)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal student: %v", err)
	}

	err = ctx.GetStub().PutState(studentKey, studentBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to PutState studentBytes %s: %v", studentBytes, err)
	}

	totalKey, err := ctx.GetStub().CreateCompositeKey(totalPrefix, []string{minter, studentId})
	if err != nil {
		return nil, fmt.Errorf("failed to CreateCompositeKey to totalkey.: %v", err)
	}

	err = ctx.GetStub().PutState(totalKey, []byte{'\u0000'})
	if err != nil {
		return nil, fmt.Errorf("failed to PutState totalKey %s: %v", studentBytes, err)
	}

	// Emit the Transferlog event
	transfer := new(Transferlog)
	transfer.From = "0x0"
	transfer.To = minter
	transfer.StudentId = studentId

	transferBytes, err := json.Marshal(transfer)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transferBytes: %v", err)
	}

	err = ctx.GetStub().SetEvent("Transfer", transferBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to SetEvent transferBytes %s: %v", transferBytes, err)
	}
	return student, nil


}

func (s *SmartContract) TransferStudent(ctx contractapi.TransactionContextInterface, from string, to string, studentId string) (bool, error){
	sender64, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return false, fmt.Errorf("failed to GetClientIdentity: %v", err)
	}

	senderBytes, err := base64.StdEncoding.DecodeString(sender64)
	if err != nil {
		return false, fmt.Errorf("failed to DecodeString sender: %v", err)
	}
	sender := string(senderBytes)
	student, err := ReadStudent(ctx, studentId)
	if err != nil {
		return false, fmt.Errorf("failed to _readStudent : %v", err)
	}

	owner := student.Owner
	if owner != sender  {
		return false, fmt.Errorf("the sender is not the current owner ")
	}

	if owner != from {
		return false, fmt.Errorf("the from is not the current owner")
	}

	student.Owner = to
	studentKey, err := ctx.GetStub().CreateCompositeKey(studentPrefix, []string{studentId})
	if err != nil {
		return false, fmt.Errorf("failed to CreateCompositeKey: %v", err)
	}

	studentBytes, err := json.Marshal(student)
	if err != nil {
		return false, fmt.Errorf("failed to marshal approval: %v", err)
	}

	err = ctx.GetStub().PutState(studentKey, studentBytes)
	if err != nil {
		return false, fmt.Errorf("failed to PutState studentBytes %s: %v", studentBytes, err)
	}

	totalKeyFrom, err := ctx.GetStub().CreateCompositeKey(totalPrefix, []string{from, studentId})
	if err != nil {
		return false, fmt.Errorf("failed to CreateCompositeKey from: %v", err)
	}

	err = ctx.GetStub().DelState(totalKeyFrom)
	if err != nil {
		return false, fmt.Errorf("failed to DelState totalKeyFrom %s: %v", studentBytes, err)
	}

	totalKeyTo, err := ctx.GetStub().CreateCompositeKey(totalPrefix, []string{to, studentId})
	if err != nil {
		return false, fmt.Errorf("failed to CreateCompositeKey to: %v", err)
	}
	err = ctx.GetStub().PutState(totalKeyTo, []byte{0})
	if err != nil {
		return false, fmt.Errorf("failed to PutState totalKeyTo %s: %v", totalKeyTo, err)
	}

	// Emit the Transferlog event
	transfer := new(Transferlog)
	transfer.From = from
	transfer.To = to
	transfer.StudentId = studentId

	transferBytes, err := json.Marshal(transfer)
	if err != nil {
		return false, fmt.Errorf("failed to marshal transferBytes: %v", err)
	}

	err = ctx.GetStub().SetEvent("Transfer", transferBytes)
	if err != nil {
		return false, fmt.Errorf("failed to SetEvent transferBytes %s: %v", transferBytes, err)
	}
	return true, nil
}

func ReadStudent(ctx contractapi.TransactionContextInterface, studentId string)(*Student, error){
	studentKey, err := ctx.GetStub().CreateCompositeKey(studentPrefix, []string {studentId})
	if err != nil {
		return nil, fmt.Errorf("failed to CreateCompositeKey %s: %v", studentId, err)
	}

	studentBytes, err := ctx.GetStub().GetState(studentKey)
	if err != nil {
		return nil, fmt.Errorf("failed to GetState %s: %v", studentId, err)
	}
	student := new(Student)
	err = json.Unmarshal(studentBytes, student)
	if err != nil {
		return nil, fmt.Errorf("failed to Unmarshal studentBytes: %v", err)
	}

	return student, nil
}
func _studentExists(ctx contractapi.TransactionContextInterface, studentId string) bool {
	studentKey, err := ctx.GetStub().CreateCompositeKey(studentPrefix, []string{studentId})
	if err != nil {
		panic("error creating CreateCompositeKey:" + err.Error())
	}

	studentBytes, err := ctx.GetStub().GetState(studentKey)
	if err != nil {
		panic("error GetState studentBytes:" + err.Error())
	}

	return len(studentBytes) > 0
}
func (s *SmartContract) TotalOf(ctx contractapi.TransactionContextInterface, owner string) int {

	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey(totalPrefix, []string{owner})
	if err != nil {
		panic("Error creating  :" + err.Error())
	}

	total := 0
	for iterator.HasNext() {
		_, err := iterator.Next()
		if err != nil {
			return 0
		}
		total++

	}
	return total
}
func (s *SmartContract) ClientAccountID(ctx contractapi.TransactionContextInterface) (string, error) {
	clientAccountID64, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to GetClientIdentity minter: %v", err)
	}

	clientAccountBytes, err := base64.StdEncoding.DecodeString(clientAccountID64)
	if err != nil {
		return "", fmt.Errorf("failed to DecodeString clientAccount64: %v", err)
	}
	clientAccount := string(clientAccountBytes)

	return clientAccount, nil
}

// func main() {
// 	Contract := new(chaincode.SmartContract)
// 	chaincode, err := contractapi.NewChaincode(Contract)
// 	if err != nil {
// 		panic("Could not create chaincode from SmartContract." + err.Error())
// 	}

// 	err = chaincode.Start()

// 	if err != nil {
// 		panic("Failed to start chaincode. " + err.Error())
// 	}
// }
func main() {
    Chaincode, err := contractapi.NewChaincode(&SmartContract{})
    if err != nil {
      log.Panicf("Error creating chaincode: %v", err)
    }

    if err := Chaincode.Start(); err != nil {
      log.Panicf("Error starting chaincode: %v", err)
    }
  }
