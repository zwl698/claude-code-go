package constants

// API Limits
// These constants define server-side limits enforced by the Anthropic API.
// Last verified: 2025-12-22

// =============================================================================
// IMAGE LIMITS
// =============================================================================

// APIImageMaxBase64Size is the maximum base64-encoded image size (API enforced).
// The API rejects images where the base64 string length exceeds this value.
// Note: This is the base64 length, NOT raw bytes. Base64 increases size by ~33%.
const APIImageMaxBase64Size = 5 * 1024 * 1024 // 5 MB

// ImageTargetRawSize is the target raw image size to stay under base64 limit after encoding.
// Base64 encoding increases size by 4/3, so we derive the max raw size:
// raw_size * 4/3 = base64_size → raw_size = base64_size * 3/4
const ImageTargetRawSize = (APIImageMaxBase64Size * 3) / 4 // 3.75 MB

// ImageMaxWidth and ImageMaxHeight are client-side maximum dimensions for image resizing.
// The API internally resizes images larger than 1568px, but this is handled server-side.
const ImageMaxWidth = 2000
const ImageMaxHeight = 2000

// =============================================================================
// PDF LIMITS
// =============================================================================

// PDFTargetRawSize is the maximum raw PDF file size that fits within the API request limit.
// The API has a 32MB total request size limit. Base64 encoding increases size by ~33%.
const PDFTargetRawSize = 20 * 1024 * 1024 // 20 MB

// APIPDFMaxPages is the maximum number of pages in a PDF accepted by the API.
const APIPDFMaxPages = 100

// PDFExtractSizeThreshold is the size threshold above which PDFs are extracted into page images.
const PDFExtractSizeThreshold = 3 * 1024 * 1024 // 3 MB

// PDFMaxExtractSize is the maximum PDF file size for the page extraction path.
const PDFMaxExtractSize = 100 * 1024 * 1024 // 100 MB

// PDFMaxPagesPerRead is the max pages the Read tool will extract in a single call.
const PDFMaxPagesPerRead = 20

// PDFAtMentionInlineThreshold is the pages threshold for inline vs reference treatment.
const PDFAtMentionInlineThreshold = 10

// =============================================================================
// MEDIA LIMITS
// =============================================================================

// APIMaxMediaPerRequest is the maximum number of media items allowed per API request.
const APIMaxMediaPerRequest = 100

// =============================================================================
// OUTPUT LIMITS
// =============================================================================

// MaxOutputSize is the maximum output size in bytes (0.25MB)
const MaxOutputSize = 0.25 * 1024 * 1024

// MaxToolResultSize is the maximum tool result size in characters
const MaxToolResultSize = 100000

// =============================================================================
// TIMEOUTS
// =============================================================================

// DefaultTimeout is the default command timeout in seconds
const DefaultTimeout = 30

// MaxStreamingTimeout is the maximum streaming timeout in seconds
const MaxStreamingTimeout = 300

// =============================================================================
// TOKEN LIMITS
// =============================================================================

// DefaultMaxTokens is the default maximum tokens for API requests
const DefaultMaxTokens = 4096

// MaxThinkingTokens is the maximum thinking tokens
const MaxThinkingTokens = 16000
