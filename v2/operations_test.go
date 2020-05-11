package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/silviucm/pgtogogen/v2/models"
	"github.com/silviucm/utils"
)

func main() {

	fmt.Println("----------------------------------------")
	fmt.Println("Starting testing of the models structure")
	fmt.Println("----------------------------------------")

	// set debug to true
	models.SetDebugMode(true)

	TestInitDatabase()

	TestInsert()
	TestInsert()
	TestInsert()
	TestInsert()
	TestInsert()
	u := TestInsert()

	TestDeleteInstance(u)

	//TestDeleteAll()

	TestDeleteWithCondition()

	TestGetById()

	lastUser, roleId := TestInsertWithRole(1)
	TestGetByMultipleIds(lastUser, roleId)

	TestTransactionWrapper()

	TestInsertsWithRoleThenSelectFromView()

	TestSelectFromMaterializedView()

	TestTransactionSelectFromMaterializedView()

	TestTransactionUpdate()

	TestUpdate()

}

func TestInitDatabase() {

	fmt.Println(" ")
	fmt.Print("Starting Database Initialization....")

	var poolMaxConnections int = 100
	_, err := models.InitDatabaseMinimal("192.168.0.1", 5432, "hellouser", "hellopass", "mydatabase", poolMaxConnections)

	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		fmt.Println("OK.")
	}
}

func TestInsert() *models.SampleUser {

	fmt.Println(" ")
	fmt.Print("Starting Static Insert....")

	var newGuid string = utils.Guid.New()

	newTestUser := &models.SampleUser{
		UserGuid:        newGuid,
		UserFirstName:   "Robinson",
		UserLastName:    "Crusoe",
		UserEnabled:     false,
		UserEmail:       "gpetrescu-" + newGuid + "@nonameemail.ca",
		UserDateCreated: time.Now(),
	}

	// control flag to indicate that the insert query will not contain the PK
	newTestUser.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence = true

	newTestUser, err := models.Tables.SampleUser.Insert(newTestUser)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		fmt.Printf("OK. New id %d \r\n", newTestUser.UserId)
	}

	return newTestUser
}

func TestInsertWithRole(roleId int32) (*models.SampleUser, int32) {

	fmt.Println(" ")
	fmt.Print("Starting Static Insert User With Role....")

	var newGuid string = utils.Guid.New()

	newTestUser := &models.SampleUser{
		UserGuid:        newGuid,
		UserFirstName:   "George",
		UserLastName:    "Mikkelsen",
		UserEnabled:     false,
		UserEmail:       "gm-" + newGuid + "@nonameemail.ca",
		UserDateCreated: time.Now(),
	}

	// control flag to indicate that the insert query will not contain the PK
	newTestUser.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence = true

	newTestUser, err := models.Tables.SampleUser.Insert(newTestUser)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		fmt.Printf("OK. New id %d \r\n", newTestUser.UserId)
	}

	fmt.Print("Insert Role for user....")
	newUserRole := &models.SampleUserRole{UrUserId: newTestUser.UserId, UrRoleId: roleId, UrDateCreated: time.Now()}
	newUserRole, err = models.Tables.SampleUserRole.Insert(newUserRole)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		fmt.Printf("OK. \r\n")
	}

	return newTestUser, roleId
}

func TestDeleteAll() {

	fmt.Println(" ")
	fmt.Print("Starting Delete All....")

	deletedRowCount, err := models.Tables.SampleUser.DeleteAll()
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		fmt.Println("OK. Number of deleted rows: %d", deletedRowCount)
	}
}

func TestDeleteInstance(user *models.SampleUser) {

	fmt.Println(" ")
	fmt.Print("Starting Delete Instance " + user.UserEmail + "....")

	isDeleted, err := models.Tables.SampleUser.DeleteInstance(user)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		fmt.Println("OK. DeleteInstance returned: %v", isDeleted)
	}
}

func TestDeleteWithCondition() {

	fmt.Println(" ")
	fmt.Print("Starting Delete With Condition....")

	userIdUnderWhichToDelete := 65

	deletedRowCount, err := models.Tables.SampleUser.Delete("user_id < $1", userIdUnderWhichToDelete)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		fmt.Printf("OK. Number of deleted rows: %d \r\n", deletedRowCount)
	}
}

func TestGetById() {

	fmt.Println(" ")

	// let's try with a real user
	var userId int32 = 80
	fmt.Printf("Starting Get By Id %d....", userId)

	existingUser, err := models.Tables.SampleUser.GetByUserId(userId)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		if existingUser != nil {
			fmt.Printf("OK. User obtained: %v \r\n", existingUser.UserEmail)
		} else {
			fmt.Println("OK. No user found for: %d \r\n", userId)
		}
	}

	// let's try with a non-existing user (please choose an id below which does not exist in the db)
	var invalidUserId int32 = 50
	fmt.Printf("Starting Get By Id %d (expecting 'No user found')....", invalidUserId)

	invalidUser, err := models.Tables.SampleUser.GetByUserId(invalidUserId)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		if invalidUser != nil {
			fmt.Printf("OK. User obtained: %v \r\n", invalidUser.UserEmail)
		} else {
			fmt.Printf("OK. No user found for: %d \r\n", invalidUserId)
		}
	}
}

func TestGetByMultipleIds(user *models.SampleUser, roleId int32) {

	fmt.Println(" ")

	var userId int32 = 80
	fmt.Printf("Starting Get By Muliple Ids %d....", userId)

	existingUserRole, err := models.Tables.SampleUserRole.GetByUrUserIdAndUrRoleId(user.UserId, roleId)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		if existingUserRole != nil {
			fmt.Printf("OK. UserRole obtained with user id %d and role id %d \r\n", existingUserRole.UrUserId, existingUserRole.UrRoleId)
		} else {
			fmt.Println("OK. No UserRole found for user id: %d and role is: %d \r\n", user.UserId, roleId)
		}
	}

}

func TestTransactionWrapper() {

	fmt.Println(" ")

	fmt.Println("Starting Transaction Wrapper...")

	var transactionFunc = func(tx *models.Transaction) error {

		tx.DeleteAllSampleUserRole()
		tx.DeleteAllSampleUser()

		var newGuid string = utils.Guid.New()
		newTestUser := &models.SampleUser{
			UserGuid:        newGuid,
			UserFirstName:   "TXGeorge",
			UserLastName:    "TXMikkelsen",
			UserEnabled:     true,
			UserEmail:       "gm-" + newGuid + "@nonameemail.ca",
			UserDateCreated: time.Now(),
		}

		// control flag to indicate that the insert query will not contain the PK
		newTestUser.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence = true

		newTxUser, err := tx.InsertSampleUser(newTestUser)
		if err != nil {
			return models.NewModelsError("insert user tx error:", err)
		}

		fmt.Printf("Ok new user created with id %d and email %v \r\n", newTxUser.UserId, newTxUser.UserEmail)

		/*
			err = tx.Tx.Rollback()
			if err != nil {
				return models.NewModelsError("rollback error:", err)
			} else {
				fmt.Printf("We performed a rollback, so you should see no user with id: %d \r\n", newTxUser.UserId)
			}
		*/

		return nil
	}

	err := models.TxWrap(transactionFunc)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		fmt.Printf("OK. \r\n")
	}

}

func TestInsertsWithRoleThenSelectFromView() {

	TestInsertWithRole(1)
	TestInsertWithRole(2)
	TestInsertWithRole(1)
	TestInsertWithRole(1)
	TestInsertWithRole(2)
	TestInsertWithRole(3)
	TestInsertWithRole(3)
	TestInsertWithRole(3)
	TestInsertWithRole(3)

	// let's select from view
	userRoles, err := models.Views.ViewUserRole.Where("role_id = $1", 1)

	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		if userRoles != nil {
			fmt.Printf("OK. We have retrieved %d records, shown below \r\n %v \r\n", len(userRoles), userRoles)
		} else {
			fmt.Printf("OK. No rows were found for this condition \r\n")
		}

	}

}

func TestSelectFromMaterializedView() {

	// let's refresh the materialized view
	refreshErr := models.Views.ViewUserRoleAdmin.RefreshMaterializedView()
	if refreshErr != nil {
		fmt.Println("Materialized View Refresh FAIL:", refreshErr.Error())
	}

	// let's select from the materialized view
	adminRoles, err := models.Views.ViewUserRoleAdmin.SelectAll()

	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		if adminRoles != nil {
			fmt.Printf("OK. Materialized View: We have retrieved %d records, shown below \r\n %v \r\n", len(adminRoles), adminRoles)
		} else {
			fmt.Printf("OK. Materialized View: No rows were found for this condition \r\n")
		}

	}

}

func TestTransactionSelectFromMaterializedView() {

	fmt.Println(" ")

	fmt.Println("Starting Transaction Wrapper...")

	var transactionFunc = func(tx *models.Transaction) error {

		var newGuid string = utils.Guid.New()
		newTestUser := &models.SampleUser{
			UserGuid:        newGuid,
			UserFirstName:   "TXVassili",
			UserLastName:    "TXRogerovici",
			UserEnabled:     true,
			UserEmail:       "gm-" + newGuid + "@nonameemail.ca",
			UserDateCreated: time.Now(),
		}

		// control flag to indicate that the insert query will not contain the PK
		newTestUser.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence = true

		newTxUser, err := tx.InsertSampleUser(newTestUser)
		if err != nil {
			return models.NewModelsError("insert user tx error:", err)
		}

		fmt.Printf("Ok new user created with id %d and email %v \r\n", newTxUser.UserId, newTxUser.UserEmail)

		newTxUserRole := new(models.SampleUserRole)
		newTxUserRole.UrRoleId = 1
		newTxUserRole.UrUserId = newTxUser.UserId
		newTxUserRole.UrDateCreated = models.Now()

		newTxUserRole, err = tx.InsertSampleUserRole(newTxUserRole)
		if err != nil {
			return models.NewModelsError("insert user role tx error:", err)
		}

		/*
			err = tx.Tx.Rollback()
			if err != nil {
				return models.NewModelsError("rollback error:", err)
			} else {
				fmt.Printf("We performed a rollback, so you should see no user with id: %d \r\n", newTxUser.UserId)
			}
		*/

		allUsers, err := tx.SelectAllSampleUserRole()
		if err != nil {
			return models.NewModelsError("SelectAllSampleUserRole tx error:", err)
		} else {
			fmt.Printf("allUserRoles count: %d \r\n", len(allUsers))
		}

		return nil
	}

	err := models.TxWrap(transactionFunc)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		fmt.Printf("OK. \r\n")
	}

}

func TestTransactionUpdate() {

	fmt.Println(" ")

	fmt.Println("Starting Update Transaction Wrapper...")

	var transactionFunc = func(tx *models.Transaction) error {

		var newGuid string = utils.Guid.New()
		newTestUser := &models.SampleUser{
			UserGuid:        newGuid,
			UserFirstName:   "TXVassili",
			UserLastName:    "TXUpdatessen",
			UserEnabled:     true,
			UserEmail:       "gm-" + newGuid + "@update.ca",
			UserDateCreated: time.Now(),
		}

		// control flag to indicate that the insert query will not contain the PK
		newTestUser.PgToGo_IgnorePKValuesWhenInsertingAndUseSequence = true

		newTxUser, err := tx.InsertSampleUser(newTestUser)
		if err != nil {
			return models.NewModelsError("insert user tx error:", err)
		}

		fmt.Printf("Ok new user created with id %d and email %v \r\n", newTxUser.UserId, newTxUser.UserEmail)

		newTxUserRole := new(models.SampleUserRole)
		newTxUserRole.UrRoleId = 1
		newTxUserRole.UrUserId = newTxUser.UserId
		newTxUserRole.UrDateCreated = models.Now()

		newTxUserRole, err = tx.InsertSampleUserRole(newTxUserRole)
		if err != nil {
			return models.NewModelsError("insert user role tx error:", err)
		}

		usersFromWhere, err := tx.WhereSampleUser("user_email LIKE $1", "%update.ca")
		if err != nil {
			return models.NewModelsError("WhereSampleUser tx error:", err)
		} else {
			fmt.Printf("Users found: %d \r\n First user's name: %v %v and email: %v \r\n", len(usersFromWhere), usersFromWhere[0].UserFirstName, usersFromWhere[0].UserLastName, usersFromWhere[0].UserEmail)
		}

		// perform update
		usersFromWhere[0].UserEmail = "heyho@gogogolangooo.com"
		usersFromWhere[0].UserLastName = "Morloni"
		nRowsUpdated, err := tx.UpdateSampleUser(&usersFromWhere[0], "user_email LIKE $8", "%update.ca")
		if err != nil {
			return models.NewModelsError("UpdateSampleUser tx error:", err)
		} else {
			fmt.Printf("Rows updated : %d \r\n", nRowsUpdated)
		}

		// verify via select all from SampleUser
		usersFromSelect, err := tx.SelectAllSampleUser()
		if err != nil {
			return models.NewModelsError("SelectAllSampleUser tx error:", err)
		} else {
			fmt.Printf("Tx Select - Users found: %d \r\n", len(usersFromSelect))
			for i := range usersFromSelect {
				fmt.Println("User Id: ", usersFromSelect[i].UserId, " --- User name: ", usersFromSelect[i].UserFirstName, " ", usersFromSelect[i].UserLastName, " - Email: ", usersFromSelect[i].UserEmail)
			}
		}

		return nil
	}

	err := models.TxWrap(transactionFunc)
	if err != nil {
		fmt.Println("FAIL:", err.Error())
	} else {
		fmt.Printf("OK. \r\n")
	}

}

func TestUpdate() {

	// perform update
	updateUserWithMask := &models.SampleUser{}
	updateUserWithMask.UserFirstName = "Giuseppe"
	updateUserWithMask.UserLastName = "Gorlami"

	nRowsUpdated, err := models.Tables.SampleUser.UpdateWithMask(updateUserWithMask, []string{"UserFirstName", "UserLastName"}, "user_last_name LIKE $3", "%Mikkelsen%")
	if err != nil {
		fmt.Println("Update SampleUser error:", err)
	} else {
		fmt.Printf("Rows updated : %d \r\n", nRowsUpdated)
	}

	// verify via select all from SampleUser
	usersFromSelect, err := models.Tables.SampleUser.SelectAll()

	// sort by user id
	//sort.Sort(models.SortSampleUserByUserId(usersFromSelect))

	// sort by last name
	sort.Sort(models.SortSampleUserByUserLastName(usersFromSelect))

	if err != nil {
		fmt.Println("SelectAll() on SampleUser  error:", err)
	} else {
		fmt.Printf("Non-tx Select - Users found: %d \r\n", len(usersFromSelect))
		for i := range usersFromSelect {
			fmt.Println("User Id: ", usersFromSelect[i].UserId, " --- User name: ", usersFromSelect[i].UserFirstName, " ", usersFromSelect[i].UserLastName, " - Email: ", usersFromSelect[i].UserEmail)
		}
	}

}
