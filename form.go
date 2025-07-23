package webx

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/tkdeng/gocrypt"
	"github.com/tkdeng/goutil"
	"github.com/tkdeng/regex"
)

type FormHandler struct {
	app         App
	sessionHash string
	formSession *goutil.CacheMap[string, map[string]string]
}

type FormCtx struct {
	fiber.Ctx
	ctx fiber.Ctx

	app     App
	Body    map[string]any
	Session *map[string]string

	formSession *goutil.CacheMap[string, map[string]string]
	session     string
	token       string
}

// NewForm creates a new form handler.
//
// This method allows easily managing session verification for user forms.
//
// Note: this method assumes forms will be submitted with the client fetch api.
func (app App) NewForm(uri string, cb func(c FormCtx) error) *FormHandler {
	sessionHash := string(goutil.URandBytes(256))
	if hash, err := gocrypt.GenerateSalt(256); err == nil {
		sessionHash = string(hash)
	}

	formSession := goutil.NewCache[string, map[string]string](10 * time.Minute)

	var smu sync.RWMutex
	sessionList := [][]byte{}

	go func() {
		time.Sleep(30 * time.Minute)

		smu.Lock()

		for i := len(sessionList) - 1; i >= 0; i-- {
			if !formSession.Has(string(sessionList[i])) {
				sessionList = append(sessionList[:i], sessionList[i+1:]...)
			}
		}

		smu.Unlock()
	}()

	app.Use(uri, app.BlockBotHeader, func(c fiber.Ctx) error {
		dID, err := deviceID(c, sessionHash)
		if err != nil {
			return app.Error(c, 400, "Bad Request!")
		}

		smu.RLock()
		session := string(goutil.URandBytes(256, &sessionList))
		smu.RUnlock()

		sessionData := map[string]string{
			"ip":       c.IP(),
			"deviceID": dID,
		}

		//todo: get browser fingerprint
		// also add alternate methods for verifying user session

		body, err := goutil.JSON.Parse(goutil.Clean(c.Body()))
		if err != nil {
			body = map[string]any{}
		}

		return cb(FormCtx{
			ctx: c,
			app: app,

			Body:    body,
			Session: &sessionData,

			formSession: formSession,
			session:     session,
			token:       string(goutil.RandBytes(64)),
		})
	})

	//todo: add client side wasm and js for form handler

	return &FormHandler{
		app:         app,
		sessionHash: sessionHash,
		formSession: formSession,
	}
}

// API verifies and continues a verified form session, and updates the token every request.
//
// Note: this method assumes forms will be submitted with the client fetch api.
// Your API should include the "session" and updated "token" on every request.
func (handler *FormHandler) API(uri string, cb func(c FormCtx) error) {
	handler.app.Use(uri, func(c fiber.Ctx) error {
		body, err := goutil.JSON.Parse(goutil.Clean(c.Body()))
		if err != nil {
			return jsonErr(c, 400, "Bad Request!")
		}

		session := goutil.ToType[string](body["session"])
		if session == "" || body["token"] == "" {
			return jsonErr(c, 400, "Bad Request!")
		}

		sessionData, err := handler.formSession.Get(session)
		if err != nil {
			return jsonErr(c, 400, "Invalid Session!")
		}

		dID, err := deviceID(c, handler.sessionHash)
		if err != nil || sessionData["deviceID"] != dID || sessionData["ip"] != c.IP() || sessionData["token"] != body["token"] {
			handler.formSession.Del(session)
			return jsonErr(c, 400, "Invalid Session!")
		}

		delete(sessionData, "token")

		//todo: get browser fingerprint
		// also add alternate methods for verifying user session

		return cb(FormCtx{
			app: handler.app,
			ctx: c,

			Body:    body,
			Session: &sessionData,

			formSession: handler.formSession,
			session:     session,
			token:       string(goutil.RandBytes(64)),
		})
	})
}

func (ctx FormCtx) Render(url string, vars ...Map) error {
	if len(vars) == 0 {
		vars = append(vars, Map{})
	}

	vars[0]["session"] = ctx.session
	vars[0]["token"] = ctx.token

	(*ctx.Session)["token"] = ctx.token
	ctx.formSession.Set(ctx.session, *ctx.Session, nil)

	return ctx.app.Render(ctx.ctx, url, vars[0])
}

func (ctx FormCtx) JSON(success bool, json ...map[string]any) error {
	if len(json) == 0 {
		json = append(json, map[string]any{})
	}

	json[0]["success"] = success
	json[0]["token"] = ctx.token

	(*ctx.Session)["token"] = ctx.token
	ctx.formSession.Set(ctx.session, *ctx.Session, nil)

	return ctx.ctx.JSON(json[0])
}

func jsonErr(c fiber.Ctx, status int, msg string) error {
	c.Status(status)
	return c.JSON(map[string]any{
		"success": false,
		"msg":     msg,
	})
}

func deviceID(c fiber.Ctx, sessionHash string) (string, error) {
	id := []byte{}

	if val, err := goutil.JSON.Stringify(c.IPs()); err == nil {
		id = goutil.Clean(val)
	} else {
		id = []byte(goutil.Clean(c.IP()))
	}

	id = append(id, regex.JoinBytes('[', goutil.Clean(c.Get("User-Agent")), ']')...)
	id = append(id, regex.JoinBytes('[', goutil.Clean(c.Get("Accept")), ']')...)
	id = append(id, regex.JoinBytes('[', goutil.Clean(c.Get("Accept-Encoding")), ']')...)
	id = append(id, regex.JoinBytes('[', goutil.Clean(c.Get("Accept-Language")), ']')...)
	id = append(id, regex.JoinBytes('[', goutil.Clean(c.Get("Connection")), ']')...)

	id = append(id, regex.JoinBytes('[', goutil.Clean(c.Get("Sec-Ch-Ua")), ']')...)
	id = append(id, regex.JoinBytes('[', goutil.Clean(c.Get("Sec-Ch-Ua-Mobile")), ']')...)
	id = append(id, regex.JoinBytes('[', goutil.Clean(c.Get("Sec-Ch-Ua-Platform")), ']')...)

	hash, err := gocrypt.FastHash(string(id), sessionHash)
	if err != nil {
		return "", err
	}

	return hash, nil
}
