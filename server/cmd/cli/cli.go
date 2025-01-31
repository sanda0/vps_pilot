package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sanda0/vps_pilot/common"
	"github.com/sanda0/vps_pilot/models"
	"github.com/sanda0/vps_pilot/utils"
)

func CreateSuperuser() {

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter email: ")
	email, _ := reader.ReadString('\n')

	fmt.Print("Enter password: ")
	password, _ := reader.ReadString('\n')

	fmt.Printf("Superuser created with email: %s and password: %s\n", email, password)

	hashedPassword, err := utils.HashString(strings.Trim(password, "\n"))
	if err != nil {
		fmt.Println("Error hashing password")
		return
	}

	user := models.User{
		Email:       strings.Trim(email, "\n"),
		Password:    string(hashedPassword),
		IsSuperuser: true,
		IsVerified:  true,
	}

	conn := common.Conn{}
	db := conn.Connect()

	result := db.Create(&user)
	if result.Error != nil {
		fmt.Println("Error creating superuser")
		return
	}

}
