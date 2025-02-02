package cli

import (
	"fmt"
	"os"
)

func CreateSuperuser() {

	// reader := bufio.NewReader(os.Stdin)

	// fmt.Print("Enter email: ")
	// email, _ := reader.ReadString('\n')

	// fmt.Print("Enter password: ")
	// password, _ := reader.ReadString('\n')

	// fmt.Printf("Superuser created with email: %s and password: %s\n", email, password)

	// hashedPassword, err := utils.HashString(strings.Trim(password, "\n"))
	// if err != nil {
	// 	fmt.Println("Error hashing password")
	// 	return
	// }

	// user := models.User{
	// 	Email:       strings.Trim(email, "\n"),
	// 	Password:    string(hashedPassword),
	// 	IsSuperuser: true,
	// 	IsVerified:  true,
	// }

	// conn := common.Conn{}
	// db := conn.Connect()

	// result := db.Create(&user)
	// if result.Error != nil {
	// 	fmt.Println("Error creating superuser")
	// 	return
	// }

}

func CreateMakeFile() error {

	// Get the database connection parameters from the environment variables or from the command line
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	fileContent := `
migration:
	@read -p "Enter migration name: " name; \
		migrate create -ext sql -dir sql/migrations $$name

migrate:
	migrate  -source file://sql/migrations \
		-database ` + fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName) + ` up

rollback:
	migrate -source file://sql/migrations \
		-database ` + fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName) + ` down

drop:
	migrate -source file://sql/migrations \
		-database ` + fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName) + ` drop

sqlc:
	sqlc generate


migratef:
	@read -p "Enter migration number: " num; \
	migrate -source file://sql/migrations \
		-database ` + fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName) + ` force $$num

		`

	fmt.Println(fileContent)
	file, err := os.Create("makefile")
	if err != nil {
		return err
	}
	defer file.Close()
	bytes := []byte(fileContent)
	file.Write(bytes)

	return nil

}
