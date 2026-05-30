package cmd

import (
	"app/auth"
	auth2 "app/core/auth"
	"app/db"
	"app/srvCtx"
	"log"

	"github.com/spf13/cobra"
)

type UserCreate struct {
	Email       string
	Forename    string
	Lastname    string
	Password    string
	Salutation  string
	IsAdmin     bool
	IsDeveloper bool
}
type UserResetPassword struct {
	Email string
}

var (
	userCreate UserCreate
)

func init() {
	createUserCmd.Flags().StringVarP(&userCreate.Email, "email", "e", "", "Email of user")
	createUserCmd.Flags().StringVarP(&userCreate.Forename, "forename", "f", "", "Forename of user")
	createUserCmd.Flags().StringVarP(&userCreate.Lastname, "lastname", "l", "", "Lastname of user")
	createUserCmd.Flags().StringVarP(&userCreate.Password, "password", "p", "", "Password of user")
	createUserCmd.Flags().StringVarP(&userCreate.Salutation, "salutation", "s", "Herr", "Salutation of user")
	createUserCmd.Flags().BoolVarP(&userCreate.IsAdmin, "is_admin", "a", false, "if specified true")
	createUserCmd.Flags().BoolVarP(&userCreate.IsDeveloper, "is_developer", "d", false, "if specified true")
	checkError(createUserCmd.MarkFlagRequired("email"))
	checkError(createUserCmd.MarkFlagRequired("forename"))
	checkError(createUserCmd.MarkFlagRequired("lastname"))
	checkError(createUserCmd.MarkFlagRequired("salutation"))

	rootCmd.AddCommand(createUserCmd)
}

var createUserCmd = &cobra.Command{
	Use:   "createUser",
	Short: "Creates a new super user in the DbTx",
	Long:  `nothing`,
	Run: func(cmd *cobra.Command, args []string) {
		/*
				./app createUser -s Herr -e udo@polder.pro -f Udo -l Polder -p Hawaii11 -a -d
			 	./app createUser -s Herr -e call-projekt@t-online.de -f Christian -l Ammon -p Hawaii11_% -a

		*/

		log.Println("Creating user:", userCreate.Salutation, userCreate.Forename, userCreate.Lastname, userCreate.Email)

		srv := srvCtx.NewLogBobCtx(nil)

		if errTx := srv.WithTx(
			func() error {
				user, err := auth2.NewAuth().CreateProfile(srv, userCreate.Email, userCreate.Password, userCreate.Salutation, userCreate.Forename, userCreate.Lastname,
					false, true, true, db.NullInt32)
				if err != nil {
					log.Fatalf("could not create user: %v\n", err)
				}

				if userCreate.IsAdmin {
					if err := auth.AddUserToGroup(user.ID, auth.GroupAdmin); err != nil {
						log.Fatalf("could not add user to admin group: %v\n", err)
					}
				}
				if userCreate.IsDeveloper {
					if err := auth.AddUserToGroup(user.ID, auth.GroupDeveloper); err != nil {
						log.Fatalf("could not add user to developer group: %v\n", err)
					}
				}
				log.Printf("Created User %s with id: %d\n", userCreate.Email, user.ID)
				return nil
			},
		); errTx != nil {
			log.Fatalf("could not create transaction:%v\n", errTx)
		}

	},
}
