//go:build ignore

package main

import (
	"fmt"
	"os"

	"go-ticketing/utils"
)

func main() {
	for _, path := range os.Args[1:] {
		tickets, err := utils.ExtractTicketsFromPDFFile(path)
		fmt.Println("FILE", path, "ERR", err)
		for _, ticket := range tickets {
			fmt.Printf("  code=%s order=%s name=%s email=%s page=%d\n", ticket.TicketCode, ticket.OrderID, ticket.Name, ticket.Email, ticket.Page)
		}
	}
}
