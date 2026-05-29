package anon

import (
	"github.com/labstack/echo/v4"
)

func Ping(c echo.Context) error {

	/*queries := models.New(db.DB)
	r, err := queries.CheckDB(dbCtx)
	if err != nil {
		r = err.Error()
		return c.JSON(http.StatusInternalServerError, r)
	}
	return c.JSON(http.StatusOK, r)
	*/
	return nil
}

func Panic(c echo.Context) error {
	panic("Zum testen")
}
