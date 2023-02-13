package simple

import (
	"fmt"

	"github.com/TAAL-GmbH/arc/api"
	apiHandler "github.com/TAAL-GmbH/arc/api/handler"
	"github.com/TAAL-GmbH/arc/api/transactionHandler"
	"github.com/labstack/echo/v4"
)

func main() {

	// Set up a basic Echo router
	e := echo.New()

	// add a single bitcoin node
	txHandler, err := transactionHandler.NewBitcoinNode("localhost", 8332, "user", "mypassword", false)
	if err != nil {
		panic(err)
	}

	// initialise the arc default api handler, with our txHandler and any handler options
	var handler api.HandlerInterface
	if handler, err = apiHandler.NewDefault(txHandler); err != nil {
		panic(err)
	}

	// Register the ARC API
	// the arc handler registers routes under /v1/...
	api.RegisterHandlers(e, handler)
	// or with a base url => /mySubDir/v1/...
	// arc.RegisterHandlersWithBaseURL(e. blocktx_api, "/arc")

	// Serve HTTP until the world ends.
	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", "0.0.0.0", 8080)))
}
