package db

import "host-credential-manager-go/go_src/models"

var hostListSeed = []models.Host{
	{
		ID:          "1",
		Hostname:    "ssh-prod-web01.internal",
		IP:          "10.0.2.14",
		Platform:    "Linux",
		Port:        "22",
		Tags:        "production,web,frontend",
		Description: "Frontend web proxy running Nginx and Node.js proxy",
		UpdatedAt:   "2026-06-20T10:30:00Z",
	},
	{
		ID:          "2",
		Hostname:    "db-mysql-master.cluster",
		IP:          "10.0.4.5",
		Platform:    "MySQL",
		Port:        "3306",
		Tags:        "production,database,replica-set",
		Description: "Primary transactional database for user accounts",
		UpdatedAt:   "2026-06-21T08:15:00Z",
	},
	{
		ID:          "3",
		Hostname:    "ad-controller.corp.local",
		IP:          "192.168.10.10",
		Platform:    "Windows",
		Port:        "389",
		Tags:        "corp,active-directory,auth",
		Description: "Active Directory Domain Controller for office employees",
		UpdatedAt:   "2026-06-19T14:20:00Z",
	},
	{
		ID:          "4",
		Hostname:    "router-core-border.net",
		IP:          "172.16.1.1",
		Platform:    "Cisco",
		Port:        "22",
		Tags:        "network,infrastructure,edge",
		Description: "Core border gateway router facing ISP upstream",
		UpdatedAt:   "2026-06-22T11:05:00Z",
	},
	{
		ID:          "5",
		Hostname:    "aws-rds-postgres.us-east-1.rds.amazonaws.com",
		IP:          "54.210.12.89",
		Platform:    "PostgreSQL",
		Port:        "5432",
		Tags:        "aws,database,production",
		Description: "AWS RDS managed instance for analytics & business reports",
		UpdatedAt:   "2026-06-23T16:40:00Z",
	},
	{
		ID:          "6",
		Hostname:    "vcenter-mgmt.local",
		IP:          "10.50.0.100",
		Platform:    "VMware",
		Port:        "443",
		Tags:        "mgmt,virtualization,vsphere",
		Description: "vCenter Server Appliance managing host clusters",
		UpdatedAt:   "2026-06-18T09:00:00Z",
	},
	{
		ID:          "7",
		Hostname:    "esxi-node-03.local",
		IP:          "10.50.0.103",
		Platform:    "VMware",
		Port:        "22",
		Tags:        "infrastructure,virtualization,hypervisor",
		Description: "Baremetal ESXi hypervisor node hosting staging VMs",
		UpdatedAt:   "2026-06-18T09:12:00Z",
	},
	{
		ID:          "8",
		Hostname:    "redis-cache-01.internal",
		IP:          "10.0.8.20",
		Platform:    "Redis",
		Port:        "6379",
		Tags:        "cache,internal,performance",
		Description: "In-memory database for application session caching",
		UpdatedAt:   "2026-06-25T12:00:00Z",
	},
	{
		ID:          "9",
		Hostname:    "windows-jumpbox.corp.local",
		IP:          "192.168.10.25",
		Platform:    "Windows",
		Port:        "3389",
		Tags:        "corp,access,security",
		Description: "RDP Jump Host for developer staging environment access",
		UpdatedAt:   "2026-06-15T15:30:00Z",
	},
	{
		ID:          "10",
		Hostname:    "git-gitlab.internal.net",
		IP:          "10.0.10.150",
		Platform:    "Linux",
		Port:        "443",
		Tags:        "devops,vcs,self-hosted",
		Description: "GitLab Enterprise self-hosted code repository manager",
		UpdatedAt:   "2026-06-24T17:15:00Z",
	},
	{
		ID:          "11",
		Hostname:    "jenkins-ci-controller",
		IP:          "10.0.10.160",
		Platform:    "Linux",
		Port:        "8080",
		Tags:        "devops,ci-cd,automation",
		Description: "Jenkins automation controller executing builds and deployments",
		UpdatedAt:   "2026-06-24T17:45:00Z",
	},
	{
		ID:          "12",
		Hostname:    "switch-floor2-core",
		IP:          "192.168.20.2",
		Platform:    "Cisco",
		Port:        "22",
		Tags:        "network,infrastructure,switch",
		Description: "Floor 2 server rack aggregation core gigabit switch",
		UpdatedAt:   "2026-06-10T11:22:00Z",
	},
	{
		ID:          "13",
		Hostname:    "mongodb-replica-01",
		IP:          "10.0.6.11",
		Platform:    "MongoDB",
		Port:        "27017",
		Tags:        "database,nosql,staging",
		Description: "MongoDB Replica Set Primary database for catalog services",
		UpdatedAt:   "2026-06-20T14:50:00Z",
	},
	{
		ID:          "14",
		Hostname:    "pve-cluster-node1",
		IP:          "10.20.0.11",
		Platform:    "Proxmox",
		Port:        "8006",
		Tags:        "virtualization,homelab,private-cloud",
		Description: "Proxmox Virtual Environment node running container clusters",
		UpdatedAt:   "2026-06-22T08:44:00Z",
	},
	{
		ID:          "15",
		Hostname:    "k8s-master-01.infra",
		IP:          "10.120.0.10",
		Platform:    "Kubernetes",
		Port:        "6443",
		Tags:        "production,orchestration,containers",
		Description: "Kubernetes control plane API master node 1",
		UpdatedAt:   "2026-06-25T11:10:00Z",
	},
	{
		ID:          "16",
		Hostname:    "k8s-worker-01.infra",
		IP:          "10.120.0.21",
		Platform:    "Kubernetes",
		Port:        "22",
		Tags:        "production,orchestration,containers",
		Description: "Kubernetes worker node 1 executing client applications",
		UpdatedAt:   "2026-06-25T11:15:00Z",
	},
	{
		ID:          "17",
		Hostname:    "nas-backup-vault",
		IP:          "192.168.1.200",
		Platform:    "FreeNAS",
		Port:        "443",
		Tags:        "storage,backup,redundant",
		Description: "TrueNAS secure storage server with automated backup pools",
		UpdatedAt:   "2026-06-12T03:00:00Z",
	},
	{
		ID:          "18",
		Hostname:    "opnsense-firewall.local",
		IP:          "192.168.1.1",
		Platform:    "OPNsense",
		Port:        "443",
		Tags:        "network,security,gateway",
		Description: "OPNsense Core edge firewall routing & VPN endpoint",
		UpdatedAt:   "2026-06-24T10:00:00Z",
	},
	{
		ID:          "19",
		Hostname:    "api-gateway.prod.cloud",
		IP:          "3.120.45.188",
		Platform:    "AWS",
		Port:        "443",
		Tags:        "production,gateway,load-balancer",
		Description: "Kong API Gateway proxying internet requests to backend microservices",
		UpdatedAt:   "2026-06-25T15:20:00Z",
	},
	{
		ID:          "20",
		Hostname:    "mac-builder-01.internal",
		IP:          "10.0.10.201",
		Platform:    "macOS",
		Port:        "22",
		Tags:        "devops,ios,runner",
		Description: "Apple Mac Mini hosting CI build agent for iOS client compiling",
		UpdatedAt:   "2026-06-14T12:00:00Z",
	},
	{
		ID:          "21",
		Hostname:    "dns-pihole.local",
		IP:          "192.168.1.5",
		Platform:    "Linux",
		Port:        "80",
		Tags:        "network,dns,internal",
		Description: "Pi-hole local DNS server, ad blocker, and local domain routing",
		UpdatedAt:   "2026-06-25T08:00:00Z",
	},
	{
		ID:          "22",
		Hostname:    "db-oracle-erp.corp",
		IP:          "10.5.1.20",
		Platform:    "Oracle",
		Port:        "1521",
		Tags:        "database,enterprise,billing",
		Description: "Oracle Database instance holding corporate ERP financials",
		UpdatedAt:   "2026-06-05T14:30:00Z",
	},
	{
		ID:          "23",
		Hostname:    "nginx-load-balancer-01",
		IP:          "10.0.2.10",
		Platform:    "Linux",
		Port:        "80",
		Tags:        "production,load-balancer,network",
		Description: "Main incoming layer 7 HTTP load balancer",
		UpdatedAt:   "2026-06-25T14:00:00Z",
	},
	{
		ID:          "24",
		Hostname:    "grafana-monitoring",
		IP:          "10.0.30.12",
		Platform:    "Grafana",
		Port:        "3000",
		Tags:        "monitoring,ops,metrics",
		Description: "Grafana analytics visualizer for system dashboard metric tracking",
		UpdatedAt:   "2026-06-25T16:00:00Z",
	},
	{
		ID:          "25",
		Hostname:    "prometheus-scraper",
		IP:          "10.0.30.10",
		Platform:    "Linux",
		Port:        "9090",
		Tags:        "monitoring,ops,telemetry",
		Description: "Prometheus server scraping cluster nodes and exporter endpoints",
		UpdatedAt:   "2026-06-25T16:10:00Z",
	},
}

var hostUserCredSeed = []models.HostCredentials{
	{
		Hostname: "ssh-prod-web01.internal",
		Userlist: []models.UserCredential{
			{Username: "ubuntu", Password: "v3ryS3cur3_web_prod_2026"},
			{Username: "admin", Password: "admin_web01_tempPass"},
			{Username: "deployer", Password: "deployer_web01_deploy!"},
		},
	},
	{
		Hostname: "db-mysql-master.cluster",
		Userlist: []models.UserCredential{
			{Username: "db_admin", Password: "my_secure_mysql_pwd_99"},
			{Username: "repl_user", Password: "repl_user_mysql_pass"},
		},
	},
	{
		Hostname: "ad-controller.corp.local",
		Userlist: []models.UserCredential{
			{Username: "Administrator", Password: "P@ssw0rd123!_AD_Ctrl"},
			{Username: "ad_audit", Password: "audit_domain_account_2026"},
		},
	},
	{
		Hostname: "router-core-border.net",
		Userlist: []models.UserCredential{
			{Username: "admin", Password: "C1sc0_R0ut3r_P@ss_Core"},
		},
	},
	{
		Hostname: "aws-rds-postgres.us-east-1.rds.amazonaws.com",
		Userlist: []models.UserCredential{
			{Username: "postgres", Password: "rds_pg_master_secret_2026"},
			{Username: "readonly_user", Password: "readonly_pg_user_2026"},
		},
	},
	{
		Hostname: "vcenter-mgmt.local",
		Userlist: []models.UserCredential{
			{Username: "administrator@vsphere.local", Password: "vChg_Mgt_2026!_Vcent"},
		},
	},
	{
		Hostname: "esxi-node-03.local",
		Userlist: []models.UserCredential{
			{Username: "root", Password: "Esxi_Node_Host_3_Secret"},
		},
	},
	{
		Hostname: "redis-cache-01.internal",
		Userlist: []models.UserCredential{
			{Username: "root", Password: "r3d1s_c@ch3_p@ss_redis"},
		},
	},
	{
		Hostname: "windows-jumpbox.corp.local",
		Userlist: []models.UserCredential{
			{Username: "jump_user", Password: "Jmp_2026_Secure_Key_RDP!"},
		},
	},
	{
		Hostname: "git-gitlab.internal.net",
		Userlist: []models.UserCredential{
			{Username: "git_admin", Password: "GtLb_Pr_2026_SuperPass!"},
		},
	},
	{
		Hostname: "jenkins-ci-controller",
		Userlist: []models.UserCredential{
			{Username: "jenkins_mgr", Password: "Jnk_Ctrl_Sec_Pass_77_CI"},
		},
	},
	{
		Hostname: "switch-floor2-core",
		Userlist: []models.UserCredential{
			{Username: "net_admin", Password: "Sw_Fl2_P@ss_Core_Switch"},
		},
	},
	{
		Hostname: "mongodb-replica-01",
		Userlist: []models.UserCredential{
			{Username: "mongo_app_user", Password: "Mng_Rep_Sec_App_2026_db"},
		},
	},
	{
		Hostname: "pve-cluster-node1",
		Userlist: []models.UserCredential{
			{Username: "root", Password: "Prxm_VE_N1_Secret_2026!"},
		},
	},
	{
		Hostname: "k8s-master-01.infra",
		Userlist: []models.UserCredential{
			{Username: "kubeadmin", Password: "K8s_Mst_1_Admin_Sec_Token"},
		},
	},
	{
		Hostname: "k8s-worker-01.infra",
		Userlist: []models.UserCredential{
			{Username: "root", Password: "K8s_Wrk_1_Sec_Key_worker"},
		},
	},
	{
		Hostname: "nas-backup-vault",
		Userlist: []models.UserCredential{
			{Username: "backup_svc", Password: "N@s_V@ult_Bkp_2026_Nas"},
		},
	},
	{
		Hostname: "opnsense-firewall.local",
		Userlist: []models.UserCredential{
			{Username: "admin", Password: "0pnSns_Fw_Admin_99_Fw"},
		},
	},
	{
		Hostname: "api-gateway.prod.cloud",
		Userlist: []models.UserCredential{
			{Username: "api_gateway_mgr", Password: "AP1_G@t3_Pr0d_Sec_Key"},
		},
	},
	{
		Hostname: "mac-builder-01.internal",
		Userlist: []models.UserCredential{
			{Username: "builder", Password: "Mc_Bld_Key_998_MacMini"},
		},
	},
	{
		Hostname: "dns-pihole.local",
		Userlist: []models.UserCredential{
			{Username: "admin", Password: "PiH0l3_DNS_AdBlock!_Pih"},
		},
	},
	{
		Hostname: "db-oracle-erp.corp",
		Userlist: []models.UserCredential{
			{Username: "sys_erp", Password: "0rcl_Erp_System_2026_Oracle"},
		},
	},
	{
		Hostname: "nginx-load-balancer-01",
		Userlist: []models.UserCredential{
			{Username: "ubuntu", Password: "Ld_Bal_Nginx_Sec_2026_!"},
		},
	},
	{
		Hostname: "grafana-monitoring",
		Userlist: []models.UserCredential{
			{Username: "admin", Password: "Grf_Mnt_P@ss_2026_Grafana"},
		},
	},
	{
		Hostname: "prometheus-scraper",
		Userlist: []models.UserCredential{
			{Username: "prometheus", Password: "Prm_Scrp_Sec_2026_Prom"},
		},
	},
}
