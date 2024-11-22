package fullname

import (
	"bufio"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xyproto/env/v2"
)

// capitalizeFirst capitalizes the first letter of a given string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

// getFullNameFromPasswd attempts to retrieve the full name from /etc/passwd
func getFullNameFromPasswd(username string) string {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) >= 5 && fields[0] == username {
			gecosField := fields[4]
			fullName := strings.Split(gecosField, ",")[0]
			return fullName
		}
	}
	return ""
}

// getFullNameFromGitConfig attempts to retrieve the full name from the ~/.gitconfig file
func getFullNameFromGitConfig() string {
	gitconfigPath := env.ExpandUser("~/.gitconfig")
	if contents, err := os.ReadFile(gitconfigPath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(contents)))
		inUserSection := false
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				section := strings.ToLower(strings.Trim(line, "[]"))
				inUserSection = (section == "user")
				continue
			}
			if inUserSection && strings.HasPrefix(strings.ToLower(line), "name") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					name := strings.TrimSpace(parts[1])
					if name != "" {
						return name
					}
				}
			}
		}
	}
	return ""
}

// getFullNameFromAccountsService attempts to retrieve the full name from AccountsService
func getFullNameFromAccountsService() string {
	accountsServicePath := filepath.Join("/var/lib/AccountsService/users", env.CurrentUser())
	if contents, err := os.ReadFile(accountsServicePath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(contents)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "FullName=") {
				fullName := strings.TrimPrefix(line, "FullName=")
				fullName = strings.TrimSpace(fullName)
				if fullName != "" {
					return fullName
				}
			}
		}
	}
	return ""
}

// Get attempts to retrieve the full name of the current user by checking multiple sources.
func Get() string {
	if u, err := user.Current(); err == nil && u.Name != "" && u.Name != u.Username {
		return u.Name
	}

	if fullName := env.StrAlt("FULLNAME", "USER_FULL_NAME"); fullName != "" {
		return fullName
	}

	if fullName := getFullNameFromGitConfig(); fullName != "" {
		return fullName
	}

	if fullName := getFullNameFromAccountsService(); fullName != "" {
		return fullName
	}

	if fullName := getFullNameFromPasswd(env.CurrentUser()); fullName != "" {
		return fullName
	}

	if userName := env.CurrentUser(); userName != "" {
		return capitalizeFirst(userName)
	}

	return ""
}
