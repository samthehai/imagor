package vipsprocessor

import (
	"github.com/davidbyttow/govips/v2/vips"
	"golang.org/x/image/colornames"
	"image/color"
	"net/url"
	"strconv"
	"strings"
)

func background(img *vips.ImageRef, w, h int, fill string) (err error) {
	fill = strings.ToLower(fill)
	if img.HasAlpha() && fill != "blur" {
		if err = img.Flatten(getColor(fill)); err != nil {
			return
		}
	}
	if fill == "black" {
		if err = img.Embed(
			(w-img.Width())/2, (h-img.Height())/2,
			w, h, vips.ExtendBlack,
		); err != nil {
			return
		}
	} else if fill == "white" {
		if err = img.Embed(
			(w-img.Width())/2, (h-img.Height())/2,
			w, h, vips.ExtendWhite,
		); err != nil {
			return
		}
	} else {
		var cp *vips.ImageRef
		cp, err = img.Copy()
		if err != nil {
			return
		}
		defer cp.Close()
		if err = cp.Thumbnail(w, h, vips.InterestingNone); err != nil {
			return
		}
		if err = img.ResizeWithVScale(
			float64(w)/float64(img.Width()), float64(h)/float64(img.Height()),
			vips.KernelLinear,
		); err != nil {
			return
		}
		if fill == "blur" {
			if err = img.GaussianBlur(50); err != nil {
				return
			}
		} else {
			// hack because no way to set background via govips
			c := getColor(fill)
			if err = img.Linear([]float64{0, 0, 0}, []float64{
				float64(c.R), float64(c.G), float64(c.B),
			}); err != nil {
				return
			}
		}
		if err = img.Composite(
			cp, vips.BlendModeOver, (w-cp.Width())/2, (h-cp.Height())/2); err != nil {
			return
		}
	}
	return
}

func watermark(img *vips.ImageRef, args []string, load func(string) ([]byte, error)) (err error) {
	ln := len(args)
	if ln < 3 {
		return
	}
	image := args[0]
	if unescape, e := url.QueryUnescape(args[0]); e == nil {
		image = unescape
	}
	var buf []byte
	if buf, err = load(image); err != nil {
		return
	}
	var overlay *vips.ImageRef
	if overlay, err = vips.NewImageFromBuffer(buf); err != nil {
		return
	}
	defer overlay.Close()
	var x, y int
	if args[1] == "center" {
		x = (img.Width() - overlay.Width()) / 2
	} else {
		x, _ = strconv.Atoi(args[1])
	}
	if args[2] == "center" {
		y = (img.Height() - overlay.Height()) / 2
	} else {
		y, _ = strconv.Atoi(args[2])
	}
	if x < 0 {
		x += img.Width() - overlay.Width()
	}
	if y < 0 {
		y += img.Height() - overlay.Height()
	}
	if ln >= 4 {
		alpha, _ := strconv.ParseFloat(args[3], 64)
		alpha /= 100
		if err = overlay.AddAlpha(); err != nil {
			return
		}
		if err = overlay.Linear([]float64{1, 1, 1, alpha}, []float64{0, 0, 0, 0}); err != nil {
			return
		}
	}
	if err = img.Composite(overlay, vips.BlendModeOver, x, y); err != nil {
		return
	}
	return
}

func getColor(name string) *vips.Color {
	vc := &vips.Color{}
	strings.TrimPrefix(strings.ToLower(name), "#")
	if c, ok := colornames.Map[strings.ToLower(name)]; ok {
		vc.R = c.R
		vc.G = c.G
		vc.B = c.B
	} else if c, ok := parseHexColor(name); ok {
		vc.R = c.R
		vc.G = c.G
		vc.B = c.B
	}
	return vc
}

func parseHexColor(s string) (c color.RGBA, ok bool) {
	c.A = 0xff
	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		return 0
	}
	switch len(s) {
	case 6:
		c.R = hexToByte(s[0])<<4 + hexToByte(s[1])
		c.G = hexToByte(s[2])<<4 + hexToByte(s[3])
		c.B = hexToByte(s[4])<<4 + hexToByte(s[5])
		ok = true
	case 3:
		c.R = hexToByte(s[0]) * 17
		c.G = hexToByte(s[1]) * 17
		c.B = hexToByte(s[2]) * 17
		ok = true
	}
	return
}