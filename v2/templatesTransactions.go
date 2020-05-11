package main

const BASE_TRANSACTIONS = `package {{.PackageName}}

/* ************************************************************* */
/* This file was automatically generated by pgtogogen.           */
/* Do not modify this file unless you know what you are doing.   */
/* ************************************************************* */

import (
	"context"
	pgx "{{.PgxImport}}"	
)

//
// DB transaction-related types and functionality
//

// Transaction isolation levels for the pgx package
const (
	IsoLevelSerializable = pgx.Serializable
	IsoLevelRepeatableRead = pgx.RepeatableRead
	IsoLevelReadCommitted = pgx.ReadCommitted
	IsoLevelReadUncommitted = pgx.ReadUncommitted	
)


// Transaction is a wrapper structure over the pgx transaction package, to avoid importing
// that package in the generated table-to-struct files.
type Transaction struct {
	Tx pgx.Tx
}

// Commit commits the current transaction
func (t *Transaction) Commit() error {
	if t.Tx == nil {
		return NewModelsErrorLocal("Transaction.Commit()", "The inner Tx transaction is nil")
	}
	return t.Tx.Commit(context.Background())
}

// Rollback attempts to rollback the current transaction
func (t *Transaction) Rollback() error {
	if t.Tx == nil {
		return NewModelsErrorLocal("Transaction.Rollback()", "The inner Tx transaction is nil")
	}
	return t.Tx.Rollback(context.Background())
}

/* BEGIN Transactions utility functions */

// TxBegin begins and returns a transaction using the default isolation level.
// Unlike TxWrap, it is the responsibility of the caller to commit and
// rollback the transaction if necessary.
func TxBegin() (*Transaction, error) {

	txWrapper := &Transaction{}
	tx, err := GetDb().Begin(context.Background())

	if err != nil {
		return nil, err
	} 
	txWrapper.Tx = tx
	return txWrapper, nil
}

// TxBeginIso begins and returns a transaction using the specified isolation level.
// The following global constants can be passed (residing in the same package):
//  IsoLevelSerializable
//  IsoLevelRepeatableRead
//  IsoLevelReadCommitted
//  IsoLevelReadUncommitted
func TxBeginIso(isolationLevel pgx.TxIsoLevel) (*Transaction, error) {

	txWrapper := &Transaction{}
	tx, err := GetDb().BeginTx(context.Background(), pgx.TxOptions{IsoLevel: isolationLevel})

	if err != nil {
		return nil, err
	} 
	txWrapper.Tx = tx
	return txWrapper, nil	
}

/*TxWrap helps wrap the transaction inside a closure function. Additional 
 arguments can be passed along to the closure via a variadic list of 
 interface{} parameters. TxWrap automatically handles commit and rollback, 
 in case of error. It returns an error in case of failure, or nil, if successful.

 Example:

	// define the transaction functionlity in this wrapper closure
	var transactionFunc = func(tx *models.Transaction, arguments ...interface{}) (interface{}, error) {

		// assuming the generated package is named models and
		// there is a TestEvent struct corresponding to a test_event table in the database
		newTestEvent := models.Tables.TestEvent.New()

		// load the event name as passed via the variadic arguments
		newTestEvent.SetEventName(arguments[0].(string))
		newTestEvent.SetEventOverview(arguments[1].(string), true)

		newTestEvent, err := tx.InsertTestEvent(newTestEvent)
		if err != nil {
			return nil, models.NewModelsError("insert event tx error:", err)
		}

		// any other transaction operations...

		// at the end, we return nil for a successful operation
		return newTestEvent, nil
	}

	// define some parameters to be passed inside the transaction
	eventName := "Donald Duck Anniversary"
	eventDescription := "Where is the party ?"

	// we defined the transaction functionality, let's run it with the event name argument
	returnedNewEvent, err := models.TxWrap(transactionFunc, eventName, eventDescription)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		if returnedNewEvent == nil {
			fmt.Printf("OK. But newlyInsertedEvent is nil \r\n")
		} else {
			// we need to make sure to convert the resulting type to the needs of this particular transaction
			fmt.Printf("OK. newlyInsertedEvent overview: " + returnedNewEvent.(*models.TestEvent).EventOverview + "  \r\n")
		}
	} */
func TxWrap(wrapperFunc func(tx *Transaction, args ...interface{}) (interface{}, error), arguments ...interface{}) (interface{}, error) {

	var errorPrefix = "TxWrap() ERROR: "

	ctx := context.Background()

	realTx, err := GetDb().Begin(ctx)
	if err != nil {
		return nil, NewModelsError(errorPrefix+"GetDb().Begin() error: ", err)
	}

	// pgx package note: Rollback is safe to call even if the tx is already closed,
	// so if the tx commits successfully, this is a no-op
	defer realTx.Rollback(ctx)

	// wrap the real tx into our wrapper
	tx := &Transaction{Tx: realTx}

	result, err := wrapperFunc(tx, arguments...)
	if err != nil {
		return nil, NewModelsError(errorPrefix+"inner wrapperFunc() error - will return and rollback: ", err)
	}

	err = realTx.Commit(ctx)
	if err != nil {
		return nil, NewModelsError(errorPrefix+"tx.Commit() error: ", err)
	}

	return result, nil
}

/* END Transactions utility functions */

`
