package constant

import (
	"os"
	"path/filepath"
)

const (
	SHOWINGS_URL              = "https://www.thespacecinema.it/api/microservice/showings/cinemas/%s/films?filmId=%s"
	SHOWINGS_URL_TODAY        = "https://www.thespacecinema.it/api/microservice/showings/cinemas/%s/films?showingDate="
	SHOWINGS_URL_TODAY_PARAMS = "T00:00:00"
	SEATS_URL                 = "https://www.thespacecinema.it/api/microservice/booking/Session/%s/%s/seats"
	CINEMAS_URL               = "https://www.thespacecinema.it/api/microservice/showings/cinemas"
	FILMS_URL                 = "https://www.thespacecinema.it/api/microservice/showings/films"

	PROXY_LIST_URL = "https://proxylist.geonode.com/api/proxy-list?limit=500&page=1&sort_by=lastChecked&sort_type=desc"
)

var (
	FilesPath string
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic("cannot determine working directory: " + err.Error())
	}
	FilesPath = filepath.Join(wd, "files")
}
