package middleware

import (
	"crypto/sha1"
	"crypto/sha256"
	b64 "encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gemfast/server/internal/config"
	"github.com/gemfast/server/internal/db"
	"github.com/gemfast/server/internal/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
)

type ChefODIC struct {
	cfg *config.Config
	db  *db.DB
}

func NewChefODIC(cfg *config.Config, db *db.DB) *ChefODIC {
	return &ChefODIC{cfg, db}
}

func (m *ChefODIC) TokenMiddlewareFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := c.Get(IdentityKey)
		if !ok {
			c.String(http.StatusInternalServerError, "Failed to get user from context")
			return
		}
		u, ok := user.(*db.User)
		if !ok {
			c.String(http.StatusInternalServerError, "Failed to cast user to db.User")
			return
		}
		auth, err := m.authUserRequest(u, c)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": err.Error()})
			return
		}
		if !auth {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid auth headers"})
			return
		}
		c.Next()
	}
}

func (m *ChefODIC) authUserRequest(u *db.User, c *gin.Context) (bool, error) {
	sign := c.GetHeader("X-Ops-Sign")
	if sign == "" {
		return false, fmt.Errorf("missing X-Ops-Sign header")
	}
	signHash := make(map[string]string)
	parts := strings.Split(sign, ";")
	for _, part := range parts {
		field := strings.Split(part, "=")
		if len(field) != 2 {
			return false, fmt.Errorf("invalid X-Ops-Sign header")
		}
		signHash[field[0]] = field[1]
	}

	if _, ok := signHash["algorithm"]; !ok {
		signHash["algorithm"] = "sha1"
	}
	candidateBlock, err := m.canonicalizeRequest(signHash["algorithm"], signHash["version"], c)
	if err != nil {
		return false, err
	}
	switch signHash["version"] {
	case "1.3":
		suppliedBlock := requestHeaderBlock(c)
		utils.VerifySignature(signHash["algorithm"], u.PublicKey, []byte(suppliedBlock), []byte(candidateBlock))
	default:
		// request_decrypted_block = @user_secret.public_decrypt(signature)
		// (request_decrypted_block == candidate_block)
	}

	return false, nil
}

func requestHeaderBlock(c *gin.Context) string {
	var authChunks []string
	for k, v := range c.Request.Header {
		if strings.HasPrefix(k, "X-Ops-Authorization-") {
			authChunks = append(authChunks, v[0])
		}
	}
	slices.Sort(authChunks)
	return strings.Join(authChunks, "\n")
}

func (m *ChefODIC) canonicalizeRequest(algorithm, version string, c *gin.Context) (string, error) {
	var canonicalizedRequestHeaders []string
	timestamp := c.GetHeader("X-Ops-Timestamp")
	serverApiVersion := c.GetHeader("X-Ops-Server-API-Version")
	userID := c.GetHeader("X-Ops-Userid")
	path := c.Request.URL.Path
	if serverApiVersion == "" {
		serverApiVersion = "0"
	}
	canonicalTime, err := time.Parse("2006-01-02T15:04:05-0700", timestamp)
	if err != nil {
		return "", fmt.Errorf("invalid X-Ops-Timestamp header")
	}
	canonicalTime = canonicalTime.UTC()
	hashedBody, err := hashedBody(algorithm, c)
	if err != nil {
		return "", err
	}
	m1 := regexp.MustCompile(`/+`)
	canonicalPath := m1.ReplaceAllString(path, "/")
	if len(canonicalPath) > 1 {
		canonicalPath = strings.TrimSuffix(canonicalPath, "/")
	}
	switch version {
	case "1.3":
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "Method:"+c.Request.Method)
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "Hashed Path:"+canonicalPath)
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "X-Ops-Content-Hash:"+hashedBody)
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "X-Ops-Sign:version="+version)
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "X-Ops-Timestamp:"+canonicalTime.String())
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "X-Ops-UserId:"+userID)
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "X-Ops-Server-API-Version:"+serverApiVersion)
		fallthrough
	case "1.0":
		fallthrough
	case "1.1":
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "Method:"+c.Request.Method)
		hashedPath, err := digestBytes(algorithm, []byte(canonicalPath))
		if err != nil {
			return "", err
		}
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "Hashed Path:"+hashedPath)
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "X-Ops-Content-Hash:"+hashedBody)
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "X-Ops-Timestamp:"+canonicalTime.String())
		canonicalUser, err := digestBytes(algorithm, []byte(userID))
		if err != nil {
			return "", err
		}
		canonicalizedRequestHeaders = append(canonicalizedRequestHeaders, "X-Ops-UserId:"+canonicalUser)

	default:
		return "", fmt.Errorf("invalid X-Ops-Sign header version")
	}

	return strings.Join(canonicalizedRequestHeaders, "\n"), nil
}

func hashedBody(algorithm string, c *gin.Context) (string, error) {
	file, err := c.FormFile("file")
	if err != nil {
		return "", err
	}
	if file != nil {
		tmpfile, err := os.CreateTemp("/tmp", "*.chef")
		if err != nil {
			return "", err
		}
		defer os.Remove(tmpfile.Name())
		if err = c.SaveUploadedFile(file, tmpfile.Name()); err != nil {
			return "", err
		}
		return digestFile(algorithm, tmpfile)
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "", err
	}
	return digestBytes(algorithm, body)
}

func digestFile(algorithm string, f *os.File) (string, error) {
	var hashedFile string
	switch algorithm {
	case "sha1":
		h := sha1.New()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
		hashedFile = b64.StdEncoding.EncodeToString(h.Sum(nil))
	case "sha256":
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
		hashedFile = b64.StdEncoding.EncodeToString(h.Sum(nil))
	default:
		return "", fmt.Errorf("invalid algorithm")
	}
	return hashedFile, nil
}

func digestBytes(algorithm string, b []byte) (string, error) {
	switch algorithm {
	case "sha1":
		h := sha1.New()
		h.Write(b)
		return b64.StdEncoding.EncodeToString(h.Sum(nil)), nil
	case "sha256":
		h := sha256.New()
		h.Write(b)
		return b64.StdEncoding.EncodeToString(h.Sum(nil)), nil
	default:
		return "", fmt.Errorf("invalid algorithm")
	}
}

// def canonical_path
// 	p = path.gsub(%r{/+}, "/")
// 	p.length > 1 ? p.chomp("/") : p
// end

// def parse_signing_description
// 	parts = signing_description.strip.split(";").inject({}) do |memo, part|
// 		field_name, field_value = part.split("=")
// 		memo[field_name.to_sym] = field_value.strip
// 		memo
// 	end
// 	Mixlib::Authentication.logger.trace "Parsed signing description: #{parts.inspect}"
// 	parts
// end
