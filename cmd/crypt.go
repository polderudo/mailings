package cmd

import (
	"app/internals"
	"log"

	"github.com/spf13/cobra"
)

type crypt struct {
	Input  string
	Key    string
	Action string
}

var (
	cryptData crypt
)

func init() {
	cryptCmd.Flags().StringVarP(&cryptData.Input, "input", "i", "", "string to crypt")
	cryptCmd.Flags().StringVarP(&cryptData.Key, "key", "k", "", "key to use, may be empty")
	cryptCmd.Flags().StringVarP(&cryptData.Action, "action", "a", "", "action to perform: d==decrypt, e==entcrypt")

	checkError(cryptCmd.MarkFlagRequired("input"))
	checkError(cryptCmd.MarkFlagRequired("action"))

	rootCmd.AddCommand(cryptCmd)
}

var cryptCmd = &cobra.Command{
	Use:   "crypt",
	Short: "encrypts or decrypts a string value",
	Long:  `nothing`,
	Run: func(cmd *cobra.Command, args []string) {
		// ./app core crypt -i "pwd 123" -a "e"
		// ./app core crypt -a "d" -i <string>

		action := "decrypt"
		if cryptData.Action == "e" {
			action = "encrypt"
		}

		log.Printf("%s :: %s\n", action, cryptData.Input)

		switch action {
		case "decrypt":
			dec, err := internals.Decrypt(cryptData.Input)
			if err != nil {
				log.Fatalf("error decrypting: %v", err)
			}
			log.Printf("decrypted string:'%s'\n", dec)
		case "encrypt":
			enc, err := internals.Encrypt(cryptData.Input)
			if err != nil {
				log.Fatalf("error encrypting: %v", err)
			}
			log.Printf("encrypted string:'%s'\n", enc)
		default:
			log.Fatalf("no action given\n")
		}
	},
}
