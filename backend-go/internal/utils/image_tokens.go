package utils

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/gif"  // 注册 GIF 解码器
	_ "image/jpeg" // 注册 JPEG 解码器
	_ "image/png"  // 注册 PNG 解码器
	"math"
	"strings"
)

// 解码器说明：这里只注册标准库支持的 GIF/JPEG/PNG 三种格式。
// WebP/AVIF 等非标准库格式 image.DecodeConfig 读不出尺寸，会回退到 imageTokenFallback
// 固定估算值——这是刻意的权衡：宁可对少数新格式给一个保守兜底，也不为此引入第三方解码依赖
// （遵循不引入新依赖的项目约定）。回退后请求仍按合理 token 计入，不会再因 base64 被当文本高估而 503。

// 图片 token 估算：沿用 Qwen3-VL 的 smart_resize 算法
// 参考: https://github.com/QwenLM/Qwen3-VL/blob/main/qwen-vl-utils/src/qwen_vl_utils/vision_process.py
//
// 算法摘要：patch=14，spatial_merge=2，所以一个 visual token 覆盖 28×28 像素；
// 图片先 smart_resize 到 [min, max] 区间（h/w 必须是 28 的倍数），
// 最终 token 数 = (H*W) / 784，并钳制到 [4, 16384]。
//
// 选 Qwen3-VL 是因为它上限 16384 比 OpenAI/Anthropic/Gemini 都高，
// 作为"保守上界"覆盖各家上游都不会低估。
//
// 计费精度提示：本模块产出的是「路由用的保守上界估算」，不是计费精度。
// 上游缺 usage 时，EstimateRequestTokens/EstimateResponsesRequestTokens 的结果会回填
// 给客户端；Qwen3-VL 的 16384 上界对 OpenAI/Anthropic 实际计费会偏高。这是刻意取舍：
// 宁可估高也别估低导致大图撞穿 scheduler 阈值被全量跳过而 503。
const (
	imagePatchFactor    = 28 // patch_size * spatial_merge_size = 14 * 2
	imageMinTokenNum    = 4
	imageMaxTokenNum    = 16384
	imageMaxAspectRatio = 200
	imageMaxDimension   = 100000 // 单边像素上限：超过即视为非法，避免超大尺寸下乘积整型溢出
	// imageHeaderSniffBytes 是从 base64 头部截取、喂给 image.DecodeConfig 的「base64 字符数」上限
	// （非解码后字节数）。base64 按 4/3 膨胀，故 90112 字符约对应 66KB 解码字节，
	// 足以覆盖 JPEG 单个 APP1(EXIF)/APP0 段（长度字段上限 64KB）把 SOF 尺寸标记推后的情况。
	// 旧值 8192 字符仅能解码约 6KB，手机照片几十 KB 的 EXIF/缩略图会把 SOF 推到其后，
	// 导致读不出尺寸而回退 imageTokenFallback（约 10× 低估）。
	imageHeaderSniffBytes = 90112 // ≈ 64KB 解码字节
	imageTokenFallback    = 1500  // WebP/AVIF/损坏图等无法读尺寸时的兜底
)

// estimateImageTokensFromSize 根据图片真实 H/W 估算 token 数。
// 输入非法（h/w <= 0、单边超过 imageMaxDimension、或长宽比 > 200）返回 fallback。
func estimateImageTokensFromSize(height, width int) int {
	if height <= 0 || width <= 0 {
		return imageTokenFallback
	}
	// sanity 上限：长宽比检查挡不住超大正方形（如 50000×50000，长宽比=1），
	// 而 hBar*wBar 这类乘积在 32 位平台会整型溢出，故在入口直接拦掉异常大的尺寸。
	if height > imageMaxDimension || width > imageMaxDimension {
		return imageTokenFallback
	}
	mx, mn := width, height
	if height > width {
		mx, mn = height, width
	}
	// 用交叉相乘代替整型除法 mx/mn > ratio，避免整型截断导致阈值附近不严格
	// （如 mx/mn=200.9 整除得 200 不触发）。mx 上界为 imageMaxDimension，
	// imageMaxAspectRatio*mn 最大约 2e7，远在 int 范围内，不会溢出。
	if mx > imageMaxAspectRatio*mn {
		return imageTokenFallback
	}

	factor := imagePatchFactor
	maxPixels := imageMaxTokenNum * factor * factor
	minPixels := imageMinTokenNum * factor * factor

	hBar := roundBy(height, factor)
	wBar := roundBy(width, factor)
	if hBar < factor {
		hBar = factor
	}
	if wBar < factor {
		wBar = factor
	}

	// 用 int64 计算像素面积：单边可达 imageMaxDimension(1e5)，hBar*wBar 最高约 1e10，
	// 在 32 位平台用 int 会溢出。int64 保证比较和后续取整在任何平台都正确。
	switch pixels := int64(hBar) * int64(wBar); {
	case pixels > int64(maxPixels):
		beta := math.Sqrt(float64(height) * float64(width) / float64(maxPixels))
		hBar = floorBy(float64(height)/beta, factor)
		wBar = floorBy(float64(width)/beta, factor)
	case pixels < int64(minPixels):
		beta := math.Sqrt(float64(minPixels) / (float64(height) * float64(width)))
		hBar = ceilBy(float64(height)*beta, factor)
		wBar = ceilBy(float64(width)*beta, factor)
	}

	if hBar < factor {
		hBar = factor
	}
	if wBar < factor {
		wBar = factor
	}

	// 缩放后 hBar*wBar 已被钳在 maxPixels 量级，但仍用 int64 保持与上面一致、杜绝任何溢出风险。
	tokens := int(int64(hBar) * int64(wBar) / int64(factor*factor))
	if tokens < imageMinTokenNum {
		return imageMinTokenNum
	}
	if tokens > imageMaxTokenNum {
		return imageMaxTokenNum
	}
	return tokens
}

func roundBy(n, factor int) int {
	return int(math.Round(float64(n)/float64(factor))) * factor
}

func floorBy(v float64, factor int) int {
	return (int(v) / factor) * factor
}

func ceilBy(v float64, factor int) int {
	return ((int(math.Ceil(v)) + factor - 1) / factor) * factor
}

// decodeImageSizeFromBase64 从 base64 字符串读取图片真实 H×W，不解码像素。
// 只对头部 imageHeaderSniffBytes 个 base64 字符做 decode（约 64KB 解码字节），
// 足以覆盖 JPEG/PNG/GIF 的尺寸字段，含手机照片几十 KB EXIF/缩略图把 SOF 推后的情况。
func decodeImageSizeFromBase64(b64 string) (height, width int, ok bool) {
	b64 = strings.TrimSpace(b64)
	if b64 == "" {
		return 0, 0, false
	}

	sniffLen := len(b64)
	if sniffLen > imageHeaderSniffBytes {
		sniffLen = imageHeaderSniffBytes
	}
	sniffLen -= sniffLen % 4 // base64 解码要求 4 字节对齐
	if sniffLen <= 0 {
		return 0, 0, false
	}

	head, err := base64.StdEncoding.DecodeString(b64[:sniffLen])
	if err != nil {
		// 容错: URL-safe / 含空白的 base64
		alt := strings.NewReplacer("-", "+", "_", "/", "\n", "", "\r", "", " ", "").Replace(b64[:sniffLen])
		alt = strings.TrimRight(alt, "=")
		if pad := len(alt) % 4; pad != 0 {
			alt += strings.Repeat("=", 4-pad)
		}
		head, err = base64.StdEncoding.DecodeString(alt)
		if err != nil {
			return 0, 0, false
		}
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(head))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, false
	}
	return cfg.Height, cfg.Width, true
}

// estimateImageTokensFromBase64 估算单张内联 base64 图片的 token 数。
// 读不到尺寸时返回 imageTokenFallback。
func estimateImageTokensFromBase64(b64 string) int {
	h, w, ok := decodeImageSizeFromBase64(b64)
	if !ok {
		return imageTokenFallback
	}
	return estimateImageTokensFromSize(h, w)
}
