package payments

import (
	"fmt"

	"github.com/conekta/conekta-go"
)

func Test() {
	const acceptLanguage = "es"
	cfg := conekta.NewConfiguration()
	client := conekta.NewAPIClient(cfg)

	fmt.Printf("client: %v\n", client)

	fmt.Println("Make payment!")
}
