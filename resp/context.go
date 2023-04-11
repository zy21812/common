package resp

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/zy21812/common/cache"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type Context struct {
	echo.Context
	Auth      *AuthInfo
	validator *validator.Validate
}
type AuthInfo struct {
	Id    int    `json:"id"`
	Uid   string `json:"uid"`
	Name  string `json:"name"`
	Role  int    `json:"role"`
	PreId string `json:"-"`
	F1    bool   `json:"f1"` //HasFee
	F2    int    `json:"f2"` //FeeLimit
}

var goCahce *cache.Memory

func init() {
	goCahce = cache.NewMemory()
}

// func R(ctx echo.Context) *Context {
// 	return &Context{ctx, nil}
// }

func (c *Context) PageOK(data interface{}, total int64) error {
	return c.JSON(http.StatusOK, &Result{Code: 200, Data: map[string]interface{}{"total": total, "list": data}})
}

func (c *Context) Ok(data interface{}) error {
	return c.JSON(http.StatusOK, &Result{Code: 200, Data: data})
}

func (c *Context) ParamErr(msg string, a ...any) error {
	return c.JSON(http.StatusOK, &Result{Code: 400, Msg: "参数解析错误:" + fmt.Sprintf(msg, a...)})
}
func (c *Context) BadRequest(msg string, a ...any) error {
	return c.JSON(http.StatusOK, &Result{Code: 400, Msg: fmt.Sprintf(msg, a...)})
}
func (c *Context) NotFound(msg string, a ...any) error {
	return c.JSON(http.StatusOK, &Result{Code: 404, Msg: fmt.Sprintf(msg, a...)})
}

func (c *Context) NoPermission() error {
	return c.JSON(http.StatusOK, &Result{Code: 403})
}

func (c *Context) NoLogin() error {
	return c.JSON(http.StatusUnauthorized, &Result{Code: 401})
}

func (c *Context) ServerErr(msg string, a ...any) error {
	return c.JSON(http.StatusOK, &Result{Code: 500, Msg: fmt.Sprintf(msg, a...)})
}
func (c *Context) Json(code int, data interface{}, msg string, a ...any) error {
	return c.JSON(http.StatusOK, &Result{Code: code, Data: data, Msg: fmt.Sprintf(msg, a...)})
}
func (c *Context) Resp(code int, data interface{}) error {
	if data != nil {
		if str, ok := data.(string); ok {
			return c.String(code, str)
		}
	}
	return c.JSON(code, data)
}
func (c *Context) Stm(buf []byte) error {
	return c.Stream(200, "application/octet-stream", bytes.NewReader(buf))
}

func (c *Context) Uri() string {
	return c.Context.Scheme() + "://" + c.Request().Host + "/"
}

func (c *Context) QueryParamInt(key string) int {
	tmp := c.QueryParam(key)
	if n, err := strconv.Atoi(tmp); err == nil {
		return n
	}
	return 0
}

func (cv *Context) BindAndValidate(i interface{}) error {
	if err := cv.Bind(i); err != nil {
		return err
	}
	logrus.Infoln("BindAndValidate", i)
	if err := cv.validator.Struct(i); err != nil {
		return err
	}
	return nil
}

func SetAuth(token string, data interface{}) {
	goCahce.SetBySliding(token, data, 10*60)
}

var anonymousUrls = []string{"/api/user.login", "/api/login"}

func Auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := &Context{c, nil, validator.New()}
		uri := c.Request().RequestURI
		if strings.HasPrefix(uri, "/api") {
			// 路由拦截 - 登录身份、资源权限判断等
			for i := range anonymousUrls {
				if strings.HasPrefix(uri, anonymousUrls[i]) {
					return next(cc)
				}
			}
			token := cc.Request().Header.Get("Authorization")
			if token != "" {
				if item := goCahce.Get(token); item != nil {
					cc.Auth = item.(*AuthInfo)
				}
			}
			if cc.Auth == nil {
				logrus.Warnf("401 [%s] %s", uri, token)
				return cc.NoLogin()
			}
			// authorization := v.(dto.Authorization)
			// if strings.EqualFold(constant.LoginToken, authorization.Type) {
			// 	if authorization.Remember {
			// 		// 记住登录有效期两周
			// 		cache.TokenManager.Set(token, authorization, cache.RememberMeExpiration)
			// 	} else {
			// 		cache.TokenManager.Set(token, authorization, cache.NotRememberExpiration)
			// 	}
			// }
			return next(cc)
		}
		return next(cc)
	}
}
