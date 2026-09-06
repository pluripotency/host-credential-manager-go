package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
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
	OS          string                  `json:"os"`
	Port        string                  `json:"port"`
	Tags        string                  `json:"tags"`
	Description string                  `json:"description"`
	Userlist    []models.UserCredential `json:"userlist"`
	Accesslist  []models.AccessItem     `json:"accesslist"`
}

type UpdateHostRequest struct {
	Hostname    *string                  `json:"hostname"`
	IP          *string                  `json:"ip"`
	Platform    *string                  `json:"platform"`
	OS          *string                  `json:"os"`
	Port        *string                  `json:"port"`
	Tags        *string                  `json:"tags"`
	Description *string                  `json:"description"`
	Userlist    *[]models.UserCredential `json:"userlist"`
	Accesslist  *[]models.AccessItem     `json:"accesslist"`
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
	api.POST("/login", loginHandler)
	api.POST("/logout", logoutHandler)
	api.GET("/role", roleHandler)
	api.GET("/ssh-fzf", getSSHFzfTargetsHandler)
	api.GET("/ssh-fzf/targets", getSSHFzfTargetsHandler)
	api.POST("/ssh-fzf/targets", getSSHFzfTargetsHandler)
	api.POST("/ssh-fzf", sshFzfHandler)
	api.GET("/targets", getSSHFzfTargetsHandler)
	api.POST("/targets", sshFzfHandler)
	api.GET("/hello", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Hello from Go!"})
	})

	// Auth routes
	adminOrUserGroup := api.Group("", RequireAuth("admin", "user"))
	adminOrUserGroup.GET("/hostlist", getHostList)
	adminOrUserGroup.GET("/password/generate", generatePasswordHandler)

	adminOnlyGroup := api.Group("", RequireAuth("admin"))
	adminOnlyGroup.POST("/hostlist", createHost)
	adminOnlyGroup.PUT("/hostlist/:id", updateHost)
	adminOnlyGroup.DELETE("/hostlist/:id", deleteHost)
	adminOnlyGroup.POST("/hostlist/import", importHosts)
	adminOnlyGroup.GET("/hostlist/export", exportHosts)
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
		OS:          strings.TrimSpace(req.OS),
		Port:        strings.TrimSpace(req.Port),
		Tags:        strings.TrimSpace(req.Tags),
		Description: strings.TrimSpace(req.Description),
		UpdatedAt:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		Accesslist:  req.Accesslist,
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
	if req.OS != nil {
		hosts[index].OS = strings.TrimSpace(*req.OS)
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
	if req.Accesslist != nil {
		hosts[index].Accesslist = *req.Accesslist
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
	var hostsToImport []models.Host
	merge := false

	contentType := c.Request().Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		fileHeader, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to get file from form data"})
		}
		src, err := fileHeader.Open()
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to open uploaded file"})
		}
		defer src.Close()

		parsedHosts, err := db.ReadHostListFromCsv(src)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Failed to parse CSV: %v", err)})
		}
		hostsToImport = parsedHosts
		merge = c.FormValue("merge") == "true"
	} else if strings.HasPrefix(contentType, "text/csv") {
		parsedHosts, err := db.ReadHostListFromCsv(c.Request().Body)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Failed to parse CSV: %v", err)})
		}
		hostsToImport = parsedHosts
		merge = c.QueryParam("merge") == "true"
	} else {
		var req ImportHostsRequest
		if err := c.Bind(&req); err != nil {
			return c.String(http.StatusBadRequest, "Invalid request body")
		}
		if req.Data == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid data format. Expected an array of records or CSV upload."})
		}
		hostsToImport = req.Data
		merge = req.Merge
	}

	nowStr := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	baseTime := time.Now().UnixNano() / int64(time.Millisecond)

	var cleaned []models.Host
	for idx, item := range hostsToImport {
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
			OS:          strings.TrimSpace(item.OS),
			Port:        strings.TrimSpace(item.Port),
			Tags:        strings.TrimSpace(item.Tags),
			Description: strings.TrimSpace(item.Description),
			UpdatedAt:   updatedAt,
			Accesslist:  item.Accesslist,
		})
	}

	if merge {
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

	return db.WriteHostListToCsv(res.Writer, hosts)
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

type SSHFzfTarget struct {
	Hostname    string `json:"hostname"`
	IP          string `json:"ip"`
	Port        string `json:"port"`
	Username    string `json:"username"`
	Protocol    string `json:"protocol"`
	Platform    string `json:"platform,omitempty"`
	OS          string `json:"os,omitempty"`
	Tags        string `json:"tags,omitempty"`
	Description string `json:"description,omitempty"`
}

func getSSHFzfTargets() ([]SSHFzfTarget, error) {
	hosts, err := db.ReadHostList()
	if err != nil {
		return nil, err
	}

	creds, err := db.ReadHostCredentials()
	if err != nil {
		return nil, err
	}

	credsMap := make(map[string][]models.UserCredential)
	for _, cr := range creds {
		credsMap[cr.Hostname] = cr.Userlist
	}

	var targets []SSHFzfTarget
	for _, host := range hosts {
		for _, access := range host.Accesslist {
			proto := strings.ToLower(strings.TrimSpace(access.Protocol))
			if proto == "ssh" || proto == "telnet" {
				port := strings.TrimSpace(access.Port)
				if port == "" {
					if proto == "telnet" {
						port = "23"
					} else {
						port = "22"
					}
				}
				users := credsMap[host.Hostname]
				if len(users) == 0 {
					targets = append(targets, SSHFzfTarget{
						Hostname:    host.Hostname,
						IP:          host.IP,
						Port:        port,
						Username:    "",
						Protocol:    proto,
						Platform:    host.Platform,
						OS:          host.OS,
						Tags:        host.Tags,
						Description: host.Description,
					})
				} else {
					for _, u := range users {
						targets = append(targets, SSHFzfTarget{
							Hostname:    host.Hostname,
							IP:          host.IP,
							Port:        port,
							Username:    u.Username,
							Protocol:    proto,
							Platform:    host.Platform,
							OS:          host.OS,
							Tags:        host.Tags,
							Description: host.Description,
						})
					}
				}
			}
		}
	}

	if targets == nil {
		targets = []SSHFzfTarget{}
	}

	return targets, nil
}

func getSSHFzfTargetsHandler(c echo.Context) error {
	targets, err := getSSHFzfTargets()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, targets)
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

	// If Hostname is empty, treat as target list request
	if req.Hostname == "" {
		targets, err := getSSHFzfTargets()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, targets)
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

var sessionSecret = make([]byte, 32)

func init() {
	_, err := rand.Read(sessionSecret)
	if err != nil {
		panic(err)
	}
}

func generateToken(role string) string {
	h := hmac.New(sha256.New, sessionSecret)
	h.Write([]byte(role))
	return fmt.Sprintf("%s.%x", role, h.Sum(nil))
}

func verifyToken(token string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", false
	}
	role := parts[0]
	expectedToken := generateToken(role)
	if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) == 1 {
		return role, true
	}
	return "", false
}

func RequireAuth(requiredRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie("session_token")
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
			}

			role, valid := verifyToken(cookie.Value)
			if !valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
			}

			if len(requiredRoles) > 0 {
				allowed := false
				for _, rr := range requiredRoles {
					if rr == role {
						allowed = true
						break
					}
				}
				if !allowed {
					return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
				}
			}

			c.Set("role", role)
			return next(c)
		}
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func loginHandler(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	conf, err := db.ReadConfig()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	var valid bool
	if req.Username == "admin" {
		valid = req.Password == conf.AdminPassword
	} else if req.Username == "user" {
		valid = req.Password == conf.UserPassword
	} else {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid username or password"})
	}

	if !valid {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid username or password"})
	}

	token := generateToken(req.Username)

	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	}
	c.SetCookie(cookie)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"role":    req.Username,
	})
}

func logoutHandler(c echo.Context) error {
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
	c.SetCookie(cookie)
	return c.JSON(http.StatusOK, map[string]bool{"success": true})
}

func roleHandler(c echo.Context) error {
	cookie, err := c.Cookie("session_token")
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{"role": nil})
	}

	role, valid := verifyToken(cookie.Value)
	if !valid {
		return c.JSON(http.StatusOK, map[string]interface{}{"role": nil})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"role": role})
}

