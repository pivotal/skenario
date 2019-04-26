package serve

import (
	"testing"

	"github.com/sclevine/agouti"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAcceptance(t *testing.T, describe spec.G, it spec.S) {
	describe("the web GUI", func() {
		var driver *agouti.WebDriver
		var page *agouti.Page
		var err error

		it.Before(func() {
			driver = agouti.ChromeDriver(agouti.ChromeOptions("args", []string{
				"--headless",
				"--allow-insecure-localhost",
				"--no-sandbox",
			}), agouti.Debug)

			err = driver.Start()
			require.NoError(t, err)

			page, err = driver.NewPage()
			require.NoError(t, err)
		})

		it("is called Skenario", func() {
			err = page.Navigate("http://localhost:3000")
			assert.NoError(t, err)

			title, err := page.Title()
			assert.NoError(t, err)
			assert.Equal(t, "Skenario", title)
		})

		it.After(func() {
			err = driver.Stop()
			assert.NoError(t, err)
		})
	})

}
