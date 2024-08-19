package web_api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
)

func Api(api *gin.RouterGroup, db *sqlx.DB) (err error) {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterStructValidation(NewDeviceStructLevelValidation, NewDevice{})
	} else {
		err = errors.New("could not register struct validators")
	}

	v1 := api.Group("/v1")

	devices := v1.Group("/devices")
	pmdResolver := NewPmdResolver(db)

	// /v1/devices/pmd
	pmd := devices.Group("/pmd")

	// /v1/devices/pmd/data
	_ = pmd.Group("/data")

	register := pmd.Group("/register")
	register.POST("/", pmdResolver.RegisterNewDeviceStatus)

	// /v1/devices/pmd/status
	status := pmd.Group("/status")
	// Update device state for a device
	status.PUT("/", func(c *gin.Context) {})
	// Retrieve device state of a device
	status.GET("/", func(c *gin.Context) {})

	return
}

type DeviceType uint

const (
	PMD DeviceType = iota
	Accessory
)

type DeviceStatus uint

const (
	Ok DeviceStatus = iota
	Off
	Suspended
)

type NewDevice struct {
	SerialId     string         `binding:"required" form:"serial_id" json:"serial_id" db:"serial_id"`
	Description  sql.NullString `binding:"required" form:"description" json:"description" db:"description"`
	DeviceType   DeviceType     `binding:"required" form:"device_type" json:"device_type" db:"device_type"`
	DeviceStatus DeviceStatus   `binding:"required" form:"device_status" json:"device_status" db:"device_status"`
}

type PmdResolver struct {
	db *sqlx.DB
}

func NewPmdResolver(db *sqlx.DB) PmdResolver {
	return PmdResolver{db}
}

func (resolver *PmdResolver) RegisterNewDeviceStatus(c *gin.Context) {
	c.Status(http.StatusOK)
}

type Device struct {
	ID uint `json:"-" db:"pk"`
	NewDevice
}

func NewDeviceStructLevelValidation(sl validator.StructLevel) {
	newDevice := sl.Current().Interface().(NewDevice)

	if len(newDevice.SerialId) < 6 {
		sl.ReportError(newDevice.SerialId, "SerialId", "serial_id", "invalid", "id is to short")
	}

	if newDevice.DeviceType < PMD || newDevice.DeviceType > Accessory {
		sl.ReportError(newDevice.DeviceType, "DeviceType", "device_type", "invalid", "outside valid values")
	}

	if newDevice.DeviceStatus < Ok || newDevice.DeviceStatus > Suspended {
		sl.ReportError(newDevice.DeviceType, "DeviceType", "device_type", "invalid", "outside valid values")
	}
}
