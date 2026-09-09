package server

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	cliGroup := api.Group("", RequireClientCertMiddleware("cert/crl.pem"))
	cliGroup.GET("/ssh-fzf", getSSHFzfTargetsHandler)
	cliGroup.GET("/ssh-fzf/targets", getSSHFzfTargetsHandler)
	cliGroup.POST("/ssh-fzf/targets", getSSHFzfTargetsHandler)
	cliGroup.POST("/ssh-fzf", sshFzfHandler)
	cliGroup.GET("/targets", getSSHFzfTargetsHandler)
	cliGroup.POST("/targets", sshFzfHandler)
	api.GET("/hello", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Hello from Go!"})
	})

	// Auth routes
	adminOrUserGroup := api.Group("", RequireAuth("admin", "user"))
	adminOrUserGroup.GET("/hostlist", getHostList)
	adminOrUserGroup.GET("/password/generate", generatePasswordHandler)
	adminOrUserGroup.GET("/client/download", downloadHcmClientHandler)

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

func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	// First pass: look for development/source tree with go.mod
	curr := dir
	for i := 0; i < 5; i++ {
		if _, errMod := os.Stat(filepath.Join(curr, "go.mod")); errMod == nil {
			if _, errHcm := os.Stat(filepath.Join(curr, "hcm-client")); errHcm == nil {
				return curr
			}
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	// Second pass: for minimal deployment/container environments without go.mod
	curr = dir
	for i := 0; i < 5; i++ {
		if _, errBuilt := os.Stat(filepath.Join(curr, "hcm-client", "built")); errBuilt == nil {
			return curr
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return "."
}

var clientBuildMutex sync.Mutex

func downloadHcmClientHandler(c echo.Context) error {
	clientBuildMutex.Lock()
	defer clientBuildMutex.Unlock()

	repoRoot := findRepoRoot()
	builtDir := filepath.Join(repoRoot, "hcm-client", "built")
	_ = os.MkdirAll(builtDir, 0755)

	binaryPath := filepath.Join(builtDir, "hcm-client")

	// If binary does not exist or certs were renewed after build, trigger build
	needBuild := false
	if binInfo, err := os.Stat(binaryPath); os.IsNotExist(err) {
		needBuild = true
	} else if caInfo, err := os.Stat(filepath.Join(repoRoot, "cert", "cacert.pem")); err == nil && caInfo.ModTime().After(binInfo.ModTime()) {
		needBuild = true
	} else if clientCertInfo, err := os.Stat(filepath.Join(repoRoot, "cert", "client_cert.pem")); err == nil && clientCertInfo.ModTime().After(binInfo.ModTime()) {
		needBuild = true
	}

	if needBuild {
		var buildErr error
		var buildOut []byte

		buildScript := filepath.Join(repoRoot, "hcm-client", "build.sh")
		if _, errScript := os.Stat(buildScript); errScript == nil {
			cmd := exec.Command(buildScript)
			cmd.Dir = repoRoot
			buildOut, buildErr = cmd.CombinedOutput()
		} else if _, errLook := exec.LookPath("go"); errLook == nil {
			// Fallback: build via go build directly
			cmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", binaryPath, "./hcm-client")
			cmd.Dir = repoRoot
			buildOut, buildErr = cmd.CombinedOutput()
		} else {
			buildErr = fmt.Errorf("neither build.sh nor go compiler found in environment")
		}

		if buildErr != nil {
			// If binary already exists (e.g. pre-built in Docker image), do not fail with 500.
			// Log a warning and proceed with the existing binary and updated cert files.
			if _, errStat := os.Stat(binaryPath); errStat == nil {
				c.Logger().Warnf("hcm-client rebuild skipped (%v: %s); serving existing binary with updated certs", buildErr, string(buildOut))
			} else {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": fmt.Sprintf("Failed to build hcm-client: %v (%s)", buildErr, string(buildOut)),
				})
			}
		}
	}

	// Verify binary exists now
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "hcm-client binary is missing and could not be built",
		})
	}

	// Ensure latest certs from repoRoot/cert are copied into builtDir/cert for packaging
	certSrcDir := filepath.Join(repoRoot, "cert")
	certDstDir := filepath.Join(builtDir, "cert")
	_ = os.MkdirAll(certDstDir, 0755)
	for _, certName := range []string{"cacert.pem", "client_cert.pem", "client_key.pem"} {
		srcPath := filepath.Join(certSrcDir, certName)
		dstPath := filepath.Join(certDstDir, certName)
		if srcData, errRead := os.ReadFile(srcPath); errRead == nil {
			_ = os.WriteFile(dstPath, srcData, 0644)
		}
	}

	// Buffer tar.gz in memory
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Walk builtDir and add files
	err := filepath.Walk(builtDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(builtDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Ensure tar entry path is clean relative path with root folder hcm-client/
		header.Name = filepath.ToSlash(filepath.Join("hcm-client", relPath))

		// Set proper executable permissions
		if info.IsDir() {
			header.Mode = 0755
		} else if strings.HasSuffix(header.Name, "hcm-client") || strings.HasSuffix(header.Name, ".sh") {
			header.Mode = 0755
		} else {
			header.Mode = 0644
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create archive: %v", err),
		})
	}

	if err := tw.Close(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if err := gw.Close(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	c.Response().Header().Set("Content-Type", "application/gzip")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="hcm-client.tgz"`)
	c.Response().Header().Set("Content-Length", strconv.Itoa(buf.Len()))

	return c.Blob(http.StatusOK, "application/gzip", buf.Bytes())
}

