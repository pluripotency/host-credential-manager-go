package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"goplur"
)

//go:embed cert/*
var embeddedCertFS embed.FS

type Target struct {
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

func (t Target) NormalizedProtocol() string {
	proto := strings.ToLower(strings.TrimSpace(t.Protocol))
	if proto == "" {
		if t.Port == "23" {
			return "telnet"
		}
		return "ssh"
	}
	return proto
}

func (t Target) ResolvedPort() int {
	p, _ := strconv.Atoi(t.Port)
	if p != 0 {
		return p
	}
	if t.NormalizedProtocol() == "telnet" {
		return 23
	}
	return 22
}

func (t Target) DisplayPlatform() string {
	parts := []string{}
	if t.Platform != "" {
		parts = append(parts, t.Platform)
	}
	if t.OS != "" && !strings.EqualFold(t.OS, t.Platform) {
		parts = append(parts, t.OS)
	}
	if len(parts) == 0 {
		return "Generic"
	}
	return strings.Join(parts, " / ")
}

func main() {
	defaultURL := os.Getenv("HCM_URL")
	if defaultURL == "" {
		defaultURL = "https://127.0.0.1:8080"
	}
	defaultCert := os.Getenv("HCM_CERT")
	defaultClientCert := os.Getenv("HCM_CLIENT_CERT")
	defaultClientKey := os.Getenv("HCM_CLIENT_KEY")
	insecureEnv := os.Getenv("HCM_INSECURE") == "1" || strings.ToLower(os.Getenv("HCM_INSECURE")) == "true"

	urlFlag := flag.String("url", defaultURL, "HCM server base URL (default: https://127.0.0.1:8080 or $HCM_URL)")
	certFlag := flag.String("cert", defaultCert, "Path to CA/server certificate (optional, uses embedded cert by default)")
	clientCertFlag := flag.String("client-cert", defaultClientCert, "Path to client certificate for mTLS (optional, uses embedded cert by default)")
	clientKeyFlag := flag.String("client-key", defaultClientKey, "Path to client private key for mTLS (optional, uses embedded key by default)")
	insecureFlag := flag.Bool("insecure", insecureEnv, "Skip TLS certificate verification")
	listFlag := flag.Bool("list", false, "Print targets list and exit")
	flag.Parse()

	serverURL := strings.TrimRight(*urlFlag, "/")
	httpClient := buildHTTPClient(*certFlag, *clientCertFlag, *clientKeyFlag, *insecureFlag)

	targets, err := fetchTargets(httpClient, serverURL)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if len(targets) == 0 {
		fmt.Println("No CLI targets (SSH or Telnet) found on HCM server.")
		return
	}

	if *listFlag {
		printTargetList(serverURL, targets)
		return
	}

	selected, err := selectHostInteractive(targets)
	if err != nil {
		fmt.Printf("\nSelection cancelled: %v\n", err)
		return
	}

	proto := selected.NormalizedProtocol()
	port := selected.ResolvedPort()

	fmt.Printf("\n[+] Target Selected: %s (%s)\n", selected.Hostname, selected.IP)
	fmt.Printf("[+] Protocol: %s, User: %s, Port: %d\n", strings.ToUpper(proto), selected.Username, port)
	if selected.DisplayPlatform() != "" {
		fmt.Printf("[+] Platform / OS: %s\n", selected.DisplayPlatform())
	}

	fmt.Print("\nEnter masterpassword for HCM server: ")
	pwdBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		log.Fatalf("Failed to read password: %v", err)
	}

	masterPassword := strings.TrimSpace(string(pwdBytes))
	if masterPassword == "" {
		log.Fatal("Master password cannot be empty.")
	}

	targetPassword, err := fetchTargetPassword(httpClient, serverURL, selected.Hostname, selected.Username, masterPassword)
	if err != nil {
		log.Fatalf("Error retrieving credentials: %v", err)
	}
	if targetPassword == "" {
		log.Fatalf("Invalid masterpassword or no credentials found for %s (%s).", selected.Hostname, selected.Username)
	}

	fmt.Println("[+] Credentials retrieved successfully. Connecting automatically...")

	switch proto {
	case "ssh":
		err = connectSSH(selected, targetPassword)
	case "telnet":
		err = connectTelnet(selected, targetPassword)
	default:
		log.Fatalf("Unsupported protocol: %s", proto)
	}

	if err != nil {
		log.Fatalf("\nConnection failed: %v", err)
	}
	fmt.Println("\n[+] Session disconnected successfully.")
}

func buildHTTPClient(certPath, clientCertPath, clientKeyPath string, insecure bool) *http.Client {
	rootCAs := x509.NewCertPool()
	loadedCA := false

	// 1. Specified cert file
	if certPath != "" {
		data, err := os.ReadFile(certPath)
		if err == nil {
			if rootCAs.AppendCertsFromPEM(data) {
				loadedCA = true
			}
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Failed to read specified cert %s: %v\n", certPath, err)
		}
	}

	// 2. Embedded cert/cacert.pem
	if !loadedCA {
		data, err := embeddedCertFS.ReadFile("cert/cacert.pem")
		if err == nil && len(data) > 0 {
			if rootCAs.AppendCertsFromPEM(data) {
				loadedCA = true
			}
		}
	}

	// 3. Fallback to local cert files on disk
	if !loadedCA {
		localCandidates := []string{
			"cert/cacert.pem",
			"../cert/cacert.pem",
			"../../cert/cacert.pem",
			"cert/cert.pem",
			"../cert/cert.pem",
		}
		for _, c := range localCandidates {
			data, err := os.ReadFile(c)
			if err == nil && len(data) > 0 {
				if rootCAs.AppendCertsFromPEM(data) {
					loadedCA = true
					break
				}
			}
		}
	}

	// Load Client Certificate for mTLS
	var clientCertificates []tls.Certificate

	// 1. Specified client cert & key flags / env vars
	if clientCertPath != "" && clientKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
		if err == nil {
			clientCertificates = []tls.Certificate{cert}
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load specified client cert/key (%s, %s): %v\n", clientCertPath, clientKeyPath, err)
		}
	}

	// 2. Embedded client_cert.pem and client_key.pem
	if len(clientCertificates) == 0 {
		certData, errCert := embeddedCertFS.ReadFile("cert/client_cert.pem")
		keyData, errKey := embeddedCertFS.ReadFile("cert/client_key.pem")
		if errCert == nil && errKey == nil && len(certData) > 0 && len(keyData) > 0 {
			cert, err := tls.X509KeyPair(certData, keyData)
			if err == nil {
				clientCertificates = []tls.Certificate{cert}
			}
		}
	}

	// 3. Fallback to local client cert files on disk
	if len(clientCertificates) == 0 {
		candidates := [][]string{
			{"cert/client_cert.pem", "cert/client_key.pem"},
			{"../cert/client_cert.pem", "../cert/client_key.pem"},
			{"../../cert/client_cert.pem", "../../cert/client_key.pem"},
		}
		for _, pair := range candidates {
			cert, err := tls.LoadX509KeyPair(pair[0], pair[1])
			if err == nil {
				clientCertificates = []tls.Certificate{cert}
				break
			}
		}
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecure,
		Certificates:       clientCertificates,
	}

	// If CA certificate is loaded, verify certificate chain against CA while ignoring hostname mismatches
	if loadedCA && !insecure {
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no certificate presented by server")
			}
			certs := make([]*x509.Certificate, len(rawCerts))
			for i, asn1Data := range rawCerts {
				cert, err := x509.ParseCertificate(asn1Data)
				if err != nil {
					return fmt.Errorf("failed to parse certificate: %w", err)
				}
				certs[i] = cert
			}
			opts := x509.VerifyOptions{
				Roots:         rootCAs,
				CurrentTime:   time.Now(),
				Intermediates: x509.NewCertPool(),
			}
			for _, cert := range certs[1:] {
				opts.Intermediates.AddCert(cert)
			}
			_, err := certs[0].Verify(opts)
			return err
		}
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
}

func fetchTargets(client *http.Client, serverURL string) ([]Target, error) {
	url := fmt.Sprintf("%s/api/ssh-fzf", serverURL)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to HCM server at %s: %w", serverURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var targets []Target
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("failed to parse targets response: %w", err)
	}

	return targets, nil
}

func fetchTargetPassword(client *http.Client, serverURL, hostname, username, masterpassword string) (string, error) {
	url := fmt.Sprintf("%s/api/ssh-fzf", serverURL)
	payload := map[string]string{
		"masterpassword": masterpassword,
		"hostname":       hostname,
		"username":       username,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := client.Post(url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to query password: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse credentials response: %w", err)
	}

	return result.Value, nil
}

func printTargetList(serverURL string, targets []Target) {
	fmt.Printf("Found %d CLI targets on %s:\n", len(targets), serverURL)
	fmt.Printf("%-28s %-8s %-14s %-20s %-20s %s\n", "HOSTNAME", "PROTOCOL", "USERNAME", "IP:PORT", "PLATFORM / OS", "TAGS")
	fmt.Println(strings.Repeat("-", 100))
	for _, t := range targets {
		ipport := fmt.Sprintf("%s:%d", t.IP, t.ResolvedPort())
		fmt.Printf("%-28s %-8s %-14s %-20s %-20s %s\n",
			t.Hostname,
			strings.ToUpper(t.NormalizedProtocol()),
			t.Username,
			ipport,
			t.DisplayPlatform(),
			t.Tags,
		)
	}
}

func selectHostInteractive(allTargets []Target) (Target, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return fallbackSelect(allTargets)
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fallbackSelect(allTargets)
	}
	defer term.Restore(fd, oldState)

	var query strings.Builder
	cursorIdx := 0

	render := func() []Target {
		fmt.Print("\r\x1b[2K\x1b[H")
		fmt.Print("\x1b[J")

		fmt.Print("=== HCM QuickConnect (Type to filter, UP/DOWN to navigate, ENTER to select, ESC/Ctrl+C to quit) ===\r\n")
		fmt.Printf("Query> %s_\r\n\r\n", query.String())

		filtered := filterTargets(allTargets, query.String())
		if len(filtered) == 0 {
			fmt.Print("  [No matching targets found]\r\n")
			return nil
		}

		if cursorIdx >= len(filtered) {
			cursorIdx = len(filtered) - 1
		}
		if cursorIdx < 0 {
			cursorIdx = 0
		}

		maxDisplay := 10
		start := 0
		if cursorIdx >= maxDisplay {
			start = cursorIdx - maxDisplay + 1
		}
		end := start + maxDisplay
		if end > len(filtered) {
			end = len(filtered)
		}

		for i := start; i < end; i++ {
			t := filtered[i]
			protoStr := strings.ToUpper(t.NormalizedProtocol())
			ipport := fmt.Sprintf("%s:%d", t.IP, t.ResolvedPort())
			prefix := "  "
			if i == cursorIdx {
				prefix = "\x1b[7m> "
			}

			line := fmt.Sprintf("%-26s | %-6s | %-18s | %-12s | %-16s | [%s]",
				t.Hostname, protoStr, ipport, t.Username, t.DisplayPlatform(), t.Tags)

			if i == cursorIdx {
				fmt.Printf("%s%s\x1b[0m\r\n", prefix, line)
			} else {
				fmt.Printf("%s%s\r\n", prefix, line)
			}
		}

		fmt.Printf("\r\nShowing %d/%d targets (Index: %d)\r\n", len(filtered), len(allTargets), cursorIdx+1)
		return filtered
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		filtered := render()

		b, err := reader.ReadByte()
		if err != nil {
			return Target{}, err
		}

		switch b {
		case 3: // Ctrl+C
			return Target{}, fmt.Errorf("interrupted")
		case 13, 10: // Enter
			if len(filtered) > 0 && cursorIdx < len(filtered) {
				term.Restore(fd, oldState)
				return filtered[cursorIdx], nil
			}
		case 127, 8: // Backspace
			str := query.String()
			if len(str) > 0 {
				query.Reset()
				query.WriteString(str[:len(str)-1])
			}
		case 27: // ESC sequence
			if reader.Buffered() >= 2 {
				b1, _ := reader.ReadByte()
				b2, _ := reader.ReadByte()
				if b1 == '[' {
					switch b2 {
					case 'A': // UP
						cursorIdx--
						if cursorIdx < 0 {
							cursorIdx = 0
						}
					case 'B': // DOWN
						cursorIdx++
						if len(filtered) > 0 && cursorIdx >= len(filtered) {
							cursorIdx = len(filtered) - 1
						}
					}
				}
			} else {
				return Target{}, fmt.Errorf("escape pressed")
			}
		case 16: // Ctrl+P
			cursorIdx--
			if cursorIdx < 0 {
				cursorIdx = 0
			}
		case 14: // Ctrl+N
			cursorIdx++
			if len(filtered) > 0 && cursorIdx >= len(filtered) {
				cursorIdx = len(filtered) - 1
			}
		default:
			if b >= 32 && b <= 126 {
				query.WriteByte(b)
				cursorIdx = 0
			}
		}
	}
}

func filterTargets(targets []Target, queryString string) []Target {
	trimmed := strings.TrimSpace(queryString)
	if trimmed == "" {
		return targets
	}

	tokens := strings.Fields(strings.ToLower(trimmed))
	var result []Target

	for _, t := range targets {
		ipport := fmt.Sprintf("%s:%d", t.IP, t.ResolvedPort())
		targetText := strings.ToLower(fmt.Sprintf("%s %s %s %s %s %s %s %s %s",
			t.Hostname, t.NormalizedProtocol(), t.IP, ipport, t.Username, t.Platform, t.OS, t.Tags, t.Description))

		matchAll := true
		for _, token := range tokens {
			if !strings.Contains(targetText, token) {
				matchAll = false
				break
			}
		}
		if matchAll {
			result = append(result, t)
		}
	}

	return result
}

func fallbackSelect(targets []Target) (Target, error) {
	fmt.Println("Select a target by number:")
	for i, t := range targets {
		proto := strings.ToUpper(t.NormalizedProtocol())
		fmt.Printf("[%d] %s (%s:%d) [%s, %s]\n", i+1, t.Hostname, t.IP, t.ResolvedPort(), proto, t.Username)
	}
	fmt.Print("Enter number: ")
	var idx int
	if _, err := fmt.Scanln(&idx); err != nil || idx < 1 || idx > len(targets) {
		return Target{}, fmt.Errorf("invalid selection")
	}
	return targets[idx-1], nil
}

func connectSSH(target Target, password string) error {
	node := goplur.NewSshNode(target.Hostname, target.IP, target.Username, password, target.Platform).
		WithDirectMode(true)
	node.SSHPort = target.ResolvedPort()

	logParams := goplur.DefaultLogParams()

	return goplur.RunSsh(node, &logParams, func(s *goplur.Session) error {
		// fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("\n Connected to %s (%s). goplur shell ready.\n", target.Hostname, target.IP)
		// fmt.Println(" Type 'exit' or press Ctrl+D to disconnect.")
		// fmt.Println("--------------------------------------------------------------------------------")
		return s.Interact(goplur.WithoutCommands())
	})
}

func connectTelnet(target Target, password string) error {
	node := goplur.NewTelnetNode(target.Hostname, target.IP, target.Username, password, target.Platform).
		WithDirectMode(true)
	node.TelnetPort = target.ResolvedPort()
	node.WithEscapeExit()

	logParams := goplur.DefaultLogParams()

	return goplur.RunTelnet(node, &logParams, func(s *goplur.Session) error {
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf(" Connected to %s (%s) via Telnet. Interactive terminal ready.\n", target.Hostname, target.IP)
		fmt.Println(" Disconnect via exit or Telnet escape (Ctrl+], then quit).")
		fmt.Println("--------------------------------------------------------------------------------")
		return s.Interact(goplur.WithoutCommands())
	})
}
