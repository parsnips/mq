package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/Khan/genqlient/graphql"
)

func PostTransaction(hook []byte) (*graphql.Response, error) {
	gql := `
mutation PostCardTransaction(
  $hook: JSON!
  $transactionId: UUID! = "uuid(this.hook.token)" @cel
  $tranCode: String! = "this.hook.type" @cel
  $cardAccount: UUID! = "uuid(this.hook.card_token)" @cel
  $amount: Decimal! = "decimal.Abs(decimal(string(this.?gpa.orValue({}).?impacted_amount.orValue(this.hook.gpa_order.jit_funding.amount))))" @cel
  $correlation: String! = "this.hook.?gpa_order.orValue({}).?token.orValue(this.hook.token)" @cel
  $skipVoid: Boolean = "this.precedingRelatedTransactionToken == null" @cel
  $precedingRelatedTransactionToken: UUID = "has(this.hook.preceding_related_transaction_token) ? this.hook.preceding_related_transaction_token : null" @cel
) {

  voidTransaction(
    id: $precedingRelatedTransactionToken
  ) @skip(if: $skipVoid) {
    transactionId
  }
 
  postTransaction(
    input: {
      transactionId: $transactionId
      tranCode: $tranCode
      params: {
        hook: $hook
        cardAccount: $cardAccount
        amount: $amount
        correlation: $correlation
      }
    }
  ) {
    transactionId 
  }
}`

	var hookJSON any
	if err := json.Unmarshal(hook, &hookJSON); err != nil {
		return nil, err
	}

	req := &graphql.Request{
		Query: gql,
		Variables: map[string]any{
			"hook": hookJSON,
		},
		OpName: "PostCardTransaction",
	}

	return Do(http.DefaultClient, req)
}

func HandleJITResponse(response *graphql.Response) int {
	fmt.Printf("jit_resp: %+v", *response)
	//Any error decline
	if len(response.Errors) > 0 {
		return 402
	}

	// Approve!
	return 200
}

func HandleWebhookResponse(response *graphql.Response) int {
	fmt.Printf("webhook_resp: %+v", *response)
	//TODO: Return error if non-unique constraint error... otherwise return 200.
	return 200
}

func Do(client *http.Client, request *graphql.Request) (*graphql.Response, error) {
	url, err := url.Parse("https://api.us-east-1.cloud.twisp.com/financial/v1/graphql")
	if err != nil {
		return nil, err
	}
	header := http.Header{}

	header.Set("authorization", fmt.Sprintf("Bearer %s", os.Getenv("SLIPLANE_OPENID_TOKEN")))
	header.Set("x-twisp-account-id", "01eac529-86c7-4186-9e56-3f0ec2005d3a")

	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req := &http.Request{
		Method: "POST",
		URL:    url,
		Header: header,
		Body:   io.NopCloser(bytes.NewReader(b)),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	respB, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response *graphql.Response

	return response, json.Unmarshal(respB, &response)
}
