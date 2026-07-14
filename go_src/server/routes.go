package server

import (
	"encoding/csv"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"host-credential-manager-go/go_src/credentials"
	"host-credential-manager-go/go_src/db"
	"host-credential-manager-go/go_src/models"
)

type CreateHostRequest struct {
	Hostname    string                  `json:"hostname"`
	IP          string                  `json:"ip"`
	Platform    string                  `json:"platform"`
	Port        string                  `json:"port"`
	Tags        string                  `json:"tags"`
	Description string                  `json:"description"`
	Userlist    []models.UserCredential `json:"userlist"`
}

type UpdateHostRequest struct {
	Hostname    *string                  `json:"hostname"`
	IP          *string                  `json:"ip"`
	Platform    *string                  `json:"platform"`
	Port        *string                  `json:"port"`
	Tags        *string                  `json:"tags"`
	Description *string                  `json:"description"`
	Userlist    *[]models.UserCredential `json:"userlist"`
}

type ImportHostsRequest struct {
	Data  []models.Host `json:"data"`
	Merge bool          `json:"merge"`
}

// RegisterRoutes registers all API routes and middlewares.
func RegisterRoutes(e *echo.Echo) {
	// IP restriction middleware
	e.Use(IPRestrictionMiddleware)

	api := e.Group("/api")
	api.GET("/hostlist", getHostList)
	api.POST("/hostlist", createHost)
	api.PUT("/hostlist/:id", updateHost)
	api.DELETE("/hostlist/:id", deleteHost)
	api.POST("/hostlist/import", importHosts)
	api.GET("/hostlist/export", exportHosts)
	api.GET("/password/generate", generatePasswordHandler)
	api.POST("/ssh-fzf", sshFzfHandler)
	api.GET("/hello", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Hello from Go!"})
	})
}

func getHostList(c echo.Context) error {
	hosts, err := db.ReadHostList()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	creds, err := db.ReadHostCredentials()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	credsMap := make(map[string][]models.UserCredential)
	for _, cr := range creds {
		credsMap[cr.Hostname] = cr.Userlist
	}

	for i := range hosts {
		if ulist, ok := credsMap[hosts[i].Hostname]; ok {
			hosts[i].Userlist = ulist
		} else {
			hosts[i].Userlist = []models.UserCredential{}
		}
	}

	return c.JSON(http.StatusOK, hosts)
}

func createHost(c echo.Context) error {
	var req CreateHostRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Invalid request body")
	}

	req.Hostname = strings.TrimSpace(req.Hostname)
	req.Platform = strings.TrimSpace(req.Platform)

	if req.Hostname == "" || req.Platform == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing required fields (hostname, platform)"})
	}

	hosts, err := db.ReadHostList()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	newID := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)

	newHost := models.Host{
		ID:          newID,
		Hostname:    req.Hostname,
		IP:          strings.TrimSpace(req.IP),
		Platform:    req.Platform,
		Port:        strings.TrimSpace(req.Port),
		Tags:        strings.TrimSpace(req.Tags),
		Description: strings.TrimSpace(req.Description),
		UpdatedAt:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}

	hosts = append([]models.Host{newHost}, hosts...)

	if err := db.WriteHostList(hosts); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	credentials, err := db.ReadHostCredentials()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	formattedUserlist := []models.UserCredential{}
	for _, u := range req.Userlist {
		formattedUserlist = append(formattedUserlist, models.UserCredential{
			Username: strings.TrimSpace(u.Username),
			Password: u.Password,
		})
	}

	found := false
	for i, cr := range credentials {
		if cr.Hostname == newHost.Hostname {
			credentials[i].Userlist = formattedUserlist
			found = true
			break
		}
	}

	if !found {
		credentials = append(credentials, models.HostCredentials{
			Hostname: newHost.Hostname,
			Userlist: formattedUserlist,
		})
	}

	if err := db.WriteHostCredentials(credentials); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	newHost.Userlist = formattedUserlist
	return c.JSON(http.StatusCreated, newHost)
}

func updateHost(c echo.Context) error {
	id := c.Param("id")

	var req UpdateHostRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Invalid request body")
	}

	hosts, err := db.ReadHostList()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	index := -1
	for i, h := range hosts {
		if h.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Host not found"})
	}

	oldHostname := hosts[index].Hostname
	newHostname := oldHostname
	if req.Hostname != nil {
		newHostname = strings.TrimSpace(*req.Hostname)
	}

	if req.Hostname != nil {
		hosts[index].Hostname = newHostname
	}
	if req.IP != nil {
		hosts[index].IP = strings.TrimSpace(*req.IP)
	}
	if req.Platform != nil {
		hosts[index].Platform = strings.TrimSpace(*req.Platform)
	}
	if req.Port != nil {
		hosts[index].Port = strings.TrimSpace(*req.Port)
	}
	if req.Tags != nil {
		hosts[index].Tags = strings.TrimSpace(*req.Tags)
	}
	if req.Description != nil {
		hosts[index].Description = strings.TrimSpace(*req.Description)
	}
	hosts[index].UpdatedAt = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	if err := db.WriteHostList(hosts); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	credentials, err := db.ReadHostCredentials()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	var formattedUserlist []models.UserCredential
	if req.Userlist != nil {
		formattedUserlist = []models.UserCredential{}
		for _, u := range *req.Userlist {
			formattedUserlist = append(formattedUserlist, models.UserCredential{
				Username: strings.TrimSpace(u.Username),
				Password: u.Password,
			})
		}
	} else {
		formattedUserlist = []models.UserCredential{}
		for _, cr := range credentials {
			if cr.Hostname == oldHostname {
				formattedUserlist = cr.Userlist
				break
			}
		}
	}

	credIndex := -1
	for i, cr := range credentials {
		if cr.Hostname == oldHostname {
			credIndex = i
			break
		}
	}

	if credIndex != -1 {
		credentials[credIndex].Hostname = newHostname
		if req.Userlist != nil {
			credentials[credIndex].Userlist = formattedUserlist
		}
	} else {
		newCredIndex := -1
		for i, cr := range credentials {
			if cr.Hostname == newHostname {
				newCredIndex = i
				break
			}
		}

		if newCredIndex != -1 {
			if req.Userlist != nil {
				credentials[newCredIndex].Userlist = formattedUserlist
			}
		} else {
			credentials = append(credentials, models.HostCredentials{
				Hostname: newHostname,
				Userlist: formattedUserlist,
			})
		}
	}

	if err := db.WriteHostCredentials(credentials); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	updatedHost := hosts[index]
	updatedHost.Userlist = formattedUserlist

	return c.JSON(http.StatusOK, updatedHost)
}

func deleteHost(c echo.Context) error {
	id := c.Param("id")

	hosts, err := db.ReadHostList()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	var hostToDelete *models.Host
	var filteredHosts []models.Host
	for _, h := range hosts {
		if h.ID == id {
			temp := h
			hostToDelete = &temp
		} else {
			filteredHosts = append(filteredHosts, h)
		}
	}

	if hostToDelete == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Host not found"})
	}

	if err := db.WriteHostList(filteredHosts); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	credentials, err := db.ReadHostCredentials()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	var filteredCreds []models.HostCredentials
	for _, cr := range credentials {
		if cr.Hostname != hostToDelete.Hostname {
			filteredCreds = append(filteredCreds, cr)
		}
	}

	if err := db.WriteHostCredentials(filteredCreds); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

func importHosts(c echo.Context) error {
	var req ImportHostsRequest
	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Invalid request body")
	}

	if req.Data == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid data format. Expected an array of records."})
	}

	nowStr := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	baseTime := time.Now().UnixNano() / int64(time.Millisecond)

	var cleaned []models.Host
	for idx, item := range req.Data {
		id := item.ID
		if id == "" {
			id = strconv.FormatInt(baseTime+int64(idx), 10)
		}

		hostname := strings.TrimSpace(item.Hostname)
		if hostname == "" {
			hostname = "unknown-host"
		}

		platform := strings.TrimSpace(item.Platform)
		if platform == "" {
			platform = "Other"
		}

		updatedAt := item.UpdatedAt
		if updatedAt == "" {
			updatedAt = nowStr
		}

		cleaned = append(cleaned, models.Host{
			ID:          id,
			Hostname:    hostname,
			IP:          strings.TrimSpace(item.IP),
			Platform:    platform,
			Port:        strings.TrimSpace(item.Port),
			Tags:        strings.TrimSpace(item.Tags),
			Description: strings.TrimSpace(item.Description),
			UpdatedAt:   updatedAt,
		})
	}

	if req.Merge {
		current, err := db.ReadHostList()
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		currentMap := make(map[string]int)
		for idx, h := range current {
			currentMap[h.Hostname] = idx
		}

		for _, h := range cleaned {
			if existingIdx, exists := currentMap[h.Hostname]; exists {
				h.ID = current[existingIdx].ID
				current[existingIdx] = h
			} else {
				current = append([]models.Host{h}, current...)
			}
		}

		if err := db.WriteHostList(current); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, map[string]interface{}{"count": len(cleaned), "merged": true})
	} else {
		if err := db.WriteHostList(cleaned); err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, map[string]interface{}{"count": len(cleaned), "merged": false})
	}
}

func exportHosts(c echo.Context) error {
	hosts, err := db.ReadHostList()
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	res := c.Response()
	res.Header().Set("Content-Type", "text/csv")
	res.Header().Set("Content-Disposition", "attachment; filename=hostlist.csv")
	res.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(res.Writer)
	defer writer.Flush()

	header := []string{"id", "hostname", "ip", "platform", "port", "tags", "description", "updatedAt"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, host := range hosts {
		record := []string{
			host.ID,
			host.Hostname,
			host.IP,
			host.Platform,
			host.Port,
			host.Tags,
			host.Description,
			host.UpdatedAt,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}

func IPRestrictionMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		conf, err := db.ReadConfig()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error")
		}

		clientIP, _, err := net.SplitHostPort(c.Request().RemoteAddr)
		if err != nil {
			clientIP = c.Request().RemoteAddr
		}

		parsedClient := net.ParseIP(clientIP)
		isLoopback := false
		if parsedClient != nil {
			isLoopback = parsedClient.IsLoopback()
		}

		allowed := false
		for _, permitted := range conf.PermitIPList {
			parsedPermitted := net.ParseIP(permitted)
			if parsedClient != nil && parsedPermitted != nil {
				if parsedClient.Equal(parsedPermitted) {
					allowed = true
					break
				}
			} else if permitted == clientIP {
				allowed = true
				break
			}

			if permitted == "127.0.0.1" && isLoopback {
				allowed = true
				break
			}
		}

		if !allowed {
			return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
		}

		return next(c)
	}
}

func generatePasswordHandler(c echo.Context) error {
	lengthStr := c.QueryParam("length")
	lowercaseStr := c.QueryParam("lowercase")
	uppercaseStr := c.QueryParam("uppercase")
	numbersStr := c.QueryParam("numbers")
	symbolsStr := c.QueryParam("symbols")

	length := 16
	if lengthStr != "" {
		if val, err := strconv.Atoi(lengthStr); err == nil {
			length = val
		}
	}

	parseBool := func(param string, defaultValue bool) bool {
		if param == "" {
			return defaultValue
		}
		return param == "true"
	}

	lowercase := parseBool(lowercaseStr, true)
	uppercase := parseBool(uppercaseStr, true)
	numbers := parseBool(numbersStr, true)
	symbols := parseBool(symbolsStr, true)

	password, strength := credentials.GeneratePassword(length, lowercase, uppercase, numbers, symbols)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"password": password,
		"strength": strength,
	})
}

type SSHFzfRequest struct {
	MasterPassword string `json:"masterpassword"`
	Hostname       string `json:"hostname"`
	Username       string `json:"username"`
}

func sshFzfHandler(c echo.Context) error {
	var req SSHFzfRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	conf, err := db.ReadConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if req.MasterPassword != conf.MasterPassword {
		return c.JSON(http.StatusOK, map[string]string{"value": ""})
	}

	creds, err := db.ReadHostCredentials()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	password := ""
	for _, cr := range creds {
		if cr.Hostname == req.Hostname {
			for _, u := range cr.Userlist {
				if u.Username == req.Username {
					password = u.Password
					break
				}
			}
			break
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"value": password})
}
