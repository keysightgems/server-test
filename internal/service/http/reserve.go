package http

import (
	// "encoding/json"
	"errors"
	"io"
	"keysight/laas/controller/internal/controller"
	"keysight/laas/controller/internal/profile"
	"keysight/laas/controller/internal/service"
	"net/http"
	"time"

	"github.com/open-traffic-generator/opentestbed/goopentestbed"
	"google.golang.org/protobuf/encoding/protojson"
)

type testbedHandler struct {
	controller TestbedController
}

type TestbedController interface {
	Routes() []Route
	Reserve(http.ResponseWriter, *http.Request)
	Release(http.ResponseWriter, *http.Request)
}

type TestbedHandler interface {
	GetController() TestbedController
	Reserve(rBody goopentestbed.Testbed, r *http.Request) (goopentestbed.ReserveResponse, error)
	Release(rBody goopentestbed.Session, r *http.Request) (goopentestbed.ReleaseResponse, error)
}

type testbedController struct {
	handler TestbedHandler
}

func NewHttpConfigurationController(handler TestbedHandler) TestbedController {
	return &testbedController{handler}
}

func NewConfigurationHandler() TestbedHandler {
	handler := new(testbedHandler)
	handler.controller = NewHttpConfigurationController(handler)
	return handler
}

func (h *testbedHandler) GetController() TestbedController {
	return h.controller
}

type ErrorDescription struct {
	Error string `json:"error"`
}

// Path: /reserve
// Method: POST
func (ctrl *testbedController) Routes() []Route {
	return []Route{
		{Path: "/reserve", Method: "POST", Name: "Reserve", Handler: ctrl.Reserve},
		{Path: "/release", Method: "POST", Name: "Release", Handler: ctrl.Release},
	}
}

func (ctrl *testbedController) Reserve(w http.ResponseWriter, r *http.Request) {
	var item goopentestbed.Testbed
	if r.Body != nil {
		body, readError := io.ReadAll(r.Body)
		if body != nil {
			item = goopentestbed.NewTestbed()
			err := item.Unmarshal().FromJson(string(body))
			if err != nil {
				ctrl.responseReserveError(w, "validation", err)
				return
			}
		} else {
			ctrl.responseReserveError(w, "validation", readError)
			return
		}
	} else {
		bodyError := errors.New("request does not have a body")
		ctrl.responseReserveError(w, "validation", bodyError)
		return
	}
	result, err := ctrl.handler.Reserve(item, r)
	if err != nil {
		ctrl.responseReserveError(w, "internal", err)
		return
	}

	if result.HasYieldResponse() {

		proto, err := result.YieldResponse().Marshal().ToProto()
		if err != nil {
			ctrl.responseReserveError(w, "validation", err)
		}
		data, err := controlMrlOpts.Marshal(proto)
		if err != nil {
			ctrl.responseReserveError(w, "validation", err)
		}
		_, err = WriteCustomJSONResponse(w, 200, data)
		if err != nil {
			ctrl.responseReserveError(w, "validation", err)
		}

		return
	}
	ctrl.responseReserveError(w, "internal", errors.New("unknown error"))
}

func (ctrl *testbedController) responseReserveError(w http.ResponseWriter, errorKind goopentestbed.ErrorKindEnum, rsp_err error) {
	var result goopentestbed.Error
	var statusCode int32
	if errorKind == "validation" {
		statusCode = 400
	} else if errorKind == "internal" {
		statusCode = 500
	}

	if rErr, ok := rsp_err.(goopentestbed.Error); ok {
		result = rErr
	} else {
		result = goopentestbed.NewError()
		err := result.Unmarshal().FromJson(rsp_err.Error())
		if err != nil {
			_ = result.SetCode(statusCode)
			err = result.SetKind(errorKind)
			if err != nil {
				log.Print(err.Error())
			}
			_ = result.SetErrors([]string{rsp_err.Error()})
		}
	}

	if _, err := WriteJSONResponse(w, int(result.Code()), result.Marshal()); err != nil {
		log.Print(err.Error())
	}
}

func (h *testbedHandler) Reserve(rBody goopentestbed.Testbed, r *http.Request) (goopentestbed.ReserveResponse, error) {
	defer profile.LogFuncDuration(time.Now(), "Reserve", "", "http")

	// validate expiry of time-limited binary
	err := service.GetTimeExpiryStatus()
	if err != nil {
		log.Error().Err(err).Msg("Reserve failed")
		return nil, err
	}

	// Call the Reserve function from the controller
	reservedResult, err := controller.Reserve(rBody)
	if err != nil {
		log.Error().Err(err).Msg("Reserve failed")
		return nil, err
	}
	result := goopentestbed.NewReserveResponse()
	result.YieldResponse().SetSessionid(reservedResult.YieldResponse().Sessionid())
	result.YieldResponse().SetTestbed(reservedResult.YieldResponse().Testbed())
	return result, nil
}

var controlMrlOpts = protojson.MarshalOptions{
	UseProtoNames:   true,
	AllowPartial:    true,
	EmitUnpopulated: true,
	Indent:          "  ",
}

func (ctrl *testbedController) Release(w http.ResponseWriter, r *http.Request) {
	var item goopentestbed.Session
	if r.Body != nil {
		body, readError := io.ReadAll(r.Body)
		if body != nil {
			item = goopentestbed.NewSession()
			err := item.Unmarshal().FromJson(string(body))
			if err != nil {
				ctrl.responseReleaseError(w, "validation", err)
				return
			}
		} else {
			ctrl.responseReleaseError(w, "validation", readError)
			return
		}
	} else {
		bodyError := errors.New("request does not have a body")
		ctrl.responseReleaseError(w, "validation", bodyError)
		return
	}
	result, err := ctrl.handler.Release(item, r)
	if err != nil {
		ctrl.responseReleaseError(w, "internal", err)
		return
	}

	if result.HasWarning() {

		proto, err := result.Warning().Marshal().ToProto()
		if err != nil {
			ctrl.responseReleaseError(w, "validation", err)
		}
		data, err := controlMrlOpts.Marshal(proto)
		if err != nil {
			ctrl.responseReleaseError(w, "validation", err)
		}
		_, err = WriteCustomJSONResponse(w, 200, data)
		if err != nil {
			ctrl.responseReleaseError(w, "validation", err)
		}

		return
	}
	ctrl.responseReleaseError(w, "internal", errors.New("unknown error"))
}

func (ctrl *testbedController) responseReleaseError(w http.ResponseWriter, errorKind goopentestbed.ErrorKindEnum, rsp_err error) {
	var result goopentestbed.Error
	var statusCode int32
	if errorKind == "validation" {
		statusCode = 400
	} else if errorKind == "internal" {
		statusCode = 500
	}

	if rErr, ok := rsp_err.(goopentestbed.Error); ok {
		result = rErr
	} else {
		result = goopentestbed.NewError()
		err := result.Unmarshal().FromJson(rsp_err.Error())
		if err != nil {
			_ = result.SetCode(statusCode)
			err = result.SetKind(errorKind)
			if err != nil {
				log.Print(err.Error())
			}
			_ = result.SetErrors([]string{rsp_err.Error()})
		}
	}

	if _, err := WriteJSONResponse(w, int(result.Code()), result.Marshal()); err != nil {
		log.Print(err.Error())
	}
}

func (h *testbedHandler) Release(rBody goopentestbed.Session, r *http.Request) (goopentestbed.ReleaseResponse, error) {
	defer profile.LogFuncDuration(time.Now(), "Release", "", "http")

	// validate expiry of time-limited binary
	err := service.GetTimeExpiryStatus()
	if err != nil {
		log.Error().Err(err).Msg("Release failed")
		return nil, err
	}

	releaseResult, err := controller.Release(rBody)
	result := goopentestbed.NewReleaseResponse()
	if err != nil {
		result.Warning().SetWarnings([]string{err.Error()})
		return result, nil
	} else {
		result.Warning().SetWarnings([]string{releaseResult.Warning().Warnings()[0]})
		return result, nil
	}
}
