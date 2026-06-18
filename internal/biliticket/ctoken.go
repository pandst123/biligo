// https://github.com/mikumifa/biliTickerBuy/pull/884/changes/31bab3557b05f42934b02a7c67a1169214e62a92
package biliticket

import (
	"encoding/base64"
	"fmt"
	"math"
	"math/rand"
	"time"
)

const (
	defaultCTokenDevicePixelRatio = 4.0
	defaultCTokenHrefFormat       = "https://mall.bilibili.com/neul-next/ticket-renovation/detail.html?id=%d"
)

var defaultCTokenHistoryLength = randIntInclusive(2, 10)

type cTokenSession struct {
	state cTokenRuntimeState
}

type cTokenRuntimeState struct {
	m1                 int
	touchend           int
	m2                 int
	visibilitychange   int
	m3                 int
	m4                 int
	openWindow         int
	m5                 int
	m6                 int
	m7                 int
	m8                 int
	m9                 int
	beforeunload       int
	ticketCollectionMs int64
	baseTimer          int
	baseTimediff       float64
	createdAtMs        int64
}

type cTokenSnapshot struct {
	m1                 int
	touchend           int
	m2                 int
	visibilitychange   int
	m3                 int
	m4                 int
	openWindow         int
	m5                 int
	timer              int
	timediff           float64
	m6                 int
	m7                 int
	m8                 int
	m9                 int
	beforeunload       int
	ticketCollectionMs int64
	baseTimer          int
}

type cTokenBrowserWindowState struct {
	scrollX           int
	scrollY           int
	innerWidth        int
	innerHeight       int
	outerWidth        int
	outerHeight       int
	screenX           int
	screenY           int
	screenWidth       int
	screenHeight      int
	screenAvailWidth  int
	screenAvailHeight int
}

type cTokenFields struct {
	m1               int
	touchend         int
	m2               int
	visibilitychange int
	m3               int
	m4               int
	openWindow       int
	m5               int
	timer            int
	timediff         float64
	m6               int
	m7               int
	m8               int
	m9               int
	beforeunload     int
}

func newCTokenSession(projectID int64, userAgent string, nowMs int64, window cTokenBrowserWindowState) cTokenSession {
	if nowMs <= 0 {
		nowMs = time.Now().UnixMilli()
	}
	return cTokenSession{
		state: initCTokenState(cTokenStateOptions{
			window:             window,
			historyLength:      defaultCTokenHistoryLength,
			userAgentLength:    len(userAgent),
			hrefLength:         len(fmt.Sprintf(defaultCTokenHrefFormat, projectID)),
			devicePixelRatio:   defaultCTokenDevicePixelRatio,
			ticketCollectionMs: nowMs,
		}),
	}
}

func (s cTokenSession) isZero() bool {
	return s.state.createdAtMs == 0
}

func (s cTokenSession) generatePrepareAt(nowMs int64) string {
	return s.state.snapshot(nowMs).generatePrepareCToken()
}

func (s cTokenSession) generateCreateAt(nowMs int64) string {
	return simCTokenState(s.state, nowMs).generateCreateCToken()
}

type cTokenStateOptions struct {
	window             cTokenBrowserWindowState
	historyLength      int
	userAgentLength    int
	hrefLength         int
	devicePixelRatio   float64
	ticketCollectionMs int64
}

func initCTokenState(options cTokenStateOptions) cTokenRuntimeState {
	if options.window.screenWidth <= 0 || options.window.screenHeight <= 0 {
		options.window = generateCTokenBrowserWindowState()
	}
	if options.historyLength <= 0 {
		options.historyLength = defaultCTokenHistoryLength
	}
	if options.userAgentLength <= 0 {
		options.userAgentLength = 140
	}
	if options.hrefLength <= 0 {
		options.hrefLength = 76
	}
	if options.devicePixelRatio <= 0 {
		options.devicePixelRatio = defaultCTokenDevicePixelRatio
	}
	if options.ticketCollectionMs <= 0 {
		options.ticketCollectionMs = time.Now().UnixMilli()
	}

	deriveD := func(index int) int {
		nowMod256 := int(time.Now().UnixMilli() % 256)
		values := []int{
			options.window.scrollX,
			options.window.scrollY,
			options.window.innerWidth,
			options.window.innerHeight,
			options.window.outerWidth,
			options.window.outerHeight,
			options.window.screenX,
			options.window.screenY,
			options.window.screenWidth,
			options.window.screenHeight,
			options.window.screenAvailWidth,
			options.historyLength,
			options.userAgentLength,
			options.hrefLength,
			int(math.Round(10 * options.devicePixelRatio)),
			nowMod256,
		}
		return (values[index%16] + values[(3*index)%16] + 17*index) & 255
	}

	return cTokenRuntimeState{
		m1:                 deriveD(1),
		touchend:           0,
		m2:                 deriveD(2),
		visibilitychange:   0,
		m3:                 deriveD(3),
		m4:                 deriveD(4),
		openWindow:         randIntInclusive(1, 3),
		m5:                 deriveD(5),
		m6:                 deriveD(6),
		m7:                 deriveD(7),
		m8:                 deriveD(8),
		m9:                 deriveD(9),
		beforeunload:       randIntInclusive(1, 3),
		ticketCollectionMs: options.ticketCollectionMs,
		baseTimer:          randIntInclusive(10, 100),
		baseTimediff:       0,
		createdAtMs:        options.ticketCollectionMs,
	}
}

func generateCTokenBrowserWindowState() cTokenBrowserWindowState {
	commonScreens := [][2]int{
		{1920, 1080},
		{2560, 1440},
		{1366, 768},
		{1440, 900},
		{1536, 864},
		{1600, 900},
		{1280, 720},
	}
	screen := commonScreens[rand.Intn(len(commonScreens))]
	screenWidth := screen[0]
	screenHeight := screen[1]
	taskbarHeight := randomChoice([]int{40, 48, 56, 64})
	screenAvailWidth := screenWidth
	screenAvailHeight := screenHeight - taskbarHeight
	maximized := rand.Float64() < 0.65
	chromeWidthDelta := randomChoice([]int{0, 8, 12, 16})
	chromeHeightDelta := randomChoice([]int{80, 88, 96, 104, 112, 120})

	var outerWidth, outerHeight, screenX, screenY int
	if maximized {
		outerWidth = screenAvailWidth
		outerHeight = screenAvailHeight
	} else {
		outerWidth = randIntInclusive(int(float64(screenAvailWidth)*0.60), int(float64(screenAvailWidth)*0.90))
		outerHeight = randIntInclusive(int(float64(screenAvailHeight)*0.60), int(float64(screenAvailHeight)*0.90))
		screenX = randIntInclusive(0, maxInt(screenAvailWidth-outerWidth, 0))
		screenY = randIntInclusive(0, maxInt(screenAvailHeight-outerHeight, 0))
	}

	return cTokenBrowserWindowState{
		scrollX:           0,
		scrollY:           0,
		innerWidth:        maxInt(320, outerWidth-chromeWidthDelta),
		innerHeight:       maxInt(240, outerHeight-chromeHeightDelta),
		outerWidth:        outerWidth,
		outerHeight:       outerHeight,
		screenX:           screenX,
		screenY:           screenY,
		screenWidth:       screenWidth,
		screenHeight:      screenHeight,
		screenAvailWidth:  screenAvailWidth,
		screenAvailHeight: screenAvailHeight,
	}
}

func (s cTokenRuntimeState) snapshot(nowMs int64) cTokenSnapshot {
	if nowMs <= 0 {
		nowMs = time.Now().UnixMilli()
	}
	elapsedSeconds := math.Max(0, float64(nowMs-s.createdAtMs)/1000)
	timediff := s.baseTimediff
	if s.ticketCollectionMs > 0 {
		timediff += math.Max(0, float64(nowMs-s.ticketCollectionMs)/1000)
	}
	return cTokenSnapshot{
		m1:                 s.m1,
		touchend:           s.touchend,
		m2:                 s.m2,
		visibilitychange:   s.visibilitychange,
		m3:                 s.m3,
		m4:                 s.m4,
		openWindow:         s.openWindow,
		m5:                 s.m5,
		timer:              s.baseTimer + int(elapsedSeconds),
		timediff:           timediff,
		m6:                 s.m6,
		m7:                 s.m7,
		m8:                 s.m8,
		m9:                 s.m9,
		beforeunload:       s.beforeunload,
		ticketCollectionMs: s.ticketCollectionMs,
		baseTimer:          s.baseTimer,
	}
}

func simCTokenState(beforeState cTokenRuntimeState, nowMs int64) cTokenSnapshot {
	if nowMs <= 0 {
		nowMs = time.Now().UnixMilli()
	}
	source := beforeState.snapshot(beforeState.createdAtMs)
	baseTimer := source.baseTimer
	if baseTimer == 0 {
		baseTimer = source.timer
	}
	touchendAdd := randomChoice([]int{0, 0, 1, 2})
	openWindowAdd := weightedZeroOrOne(80)
	visibilitychangeAdd := weightedZeroOrOne(80)
	elapsedSeconds := float64(nowMs-source.ticketCollectionMs) / 1000

	return cTokenSnapshot{
		m1:                 source.m1,
		touchend:           source.touchend + touchendAdd,
		m2:                 source.m2,
		visibilitychange:   source.visibilitychange + visibilitychangeAdd,
		m3:                 source.m3,
		m4:                 source.m4,
		openWindow:         source.openWindow + openWindowAdd,
		m5:                 source.m5,
		timer:              baseTimer + int(elapsedSeconds),
		timediff:           math.Max(0, elapsedSeconds),
		m6:                 source.m6,
		m7:                 source.m7,
		m8:                 source.m8,
		m9:                 source.m9,
		beforeunload:       -1,
		ticketCollectionMs: source.ticketCollectionMs,
		baseTimer:          baseTimer,
	}
}

func (s cTokenSnapshot) generatePrepareCToken() string {
	return generateCToken(cTokenFields{
		m1:               s.m1,
		touchend:         s.touchend,
		m2:               s.m2,
		visibilitychange: s.visibilitychange,
		m3:               s.m3,
		m4:               s.m4,
		openWindow:       s.openWindow,
		m5:               s.m5,
		timer:            s.timer,
		timediff:         s.timediff,
		m6:               s.m6,
		m7:               s.m7,
		m8:               s.m8,
		m9:               s.m9,
		beforeunload:     s.beforeunload,
	})
}

func (s cTokenSnapshot) generateCreateCToken() string {
	return generateCToken(cTokenFields{
		m1:               s.m1,
		touchend:         s.touchend,
		m2:               s.m2,
		visibilitychange: s.visibilitychange,
		m3:               s.m3,
		m4:               s.m4,
		openWindow:       -1,
		m5:               s.m5,
		timer:            s.timer,
		timediff:         s.timediff,
		m6:               s.m6,
		m7:               s.m7,
		m8:               s.m8,
		m9:               s.m9,
		beforeunload:     -1,
	})
}

func generateCToken(fields cTokenFields) string {
	if fields.touchend == -1 {
		fields.touchend = randIntInclusive(30, 50)
	}
	if fields.visibilitychange == -1 {
		fields.visibilitychange = randIntInclusive(10, 50)
	}
	if fields.beforeunload == -1 {
		if fields.openWindow != -1 {
			fields.beforeunload = fields.openWindow
		} else {
			fields.beforeunload = randIntInclusive(10, 50)
		}
	}
	if fields.timer == -1 {
		fields.timer = randIntInclusive(1, 10)
	}

	semantic := []byte{
		jsByte(fields.m1),
		jsByte(fields.touchend),
		jsByte(fields.m2),
		jsByte(fields.visibilitychange),
		jsByte(fields.m3),
		jsByte(fields.m4),
		jsByte(fields.beforeunload),
		jsByte(fields.m5),
		byte(jsUint16(fields.timer) >> 8),
		byte(jsUint16(fields.timer)),
		byte(jsUint16(int(fields.timediff)) >> 8),
		byte(jsUint16(int(fields.timediff))),
		jsByte(fields.m6),
		jsByte(fields.m7),
		jsByte(fields.m8),
		jsByte(fields.m9),
	}
	transport := make([]byte, 0, len(semantic)*2)
	for _, value := range semantic {
		transport = append(transport, value, 0)
	}
	return base64.StdEncoding.EncodeToString(transport)
}

func jsByte(value int) byte {
	if value < 0 || value > 255 {
		return 0xff
	}
	return byte(value)
}

func jsUint16(value int) uint16 {
	if value < 0 || value > 65535 {
		return 0xffff
	}
	return uint16(value)
}

func randIntInclusive(minValue int, maxValue int) int {
	if maxValue < minValue {
		return minValue
	}
	return rand.Intn(maxValue-minValue+1) + minValue
}

func randomChoice(values []int) int {
	if len(values) == 0 {
		return 0
	}
	return values[rand.Intn(len(values))]
}

func weightedZeroOrOne(zeroPercent int) int {
	if rand.Intn(100) < zeroPercent {
		return 0
	}
	return 1
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
