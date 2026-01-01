package gobackend

import (
	"strings"
	"unicode"
)

// Japanese character ranges
const (
	hiraganaStart = 0x3040
	hiraganaEnd   = 0x309F
	katakanaStart = 0x30A0
	katakanaEnd   = 0x30FF
	kanjiStart    = 0x4E00
	kanjiEnd      = 0x9FFF
)

// hiraganaToRomaji maps hiragana characters to romaji
var hiraganaToRomaji = map[rune]string{
	// Basic vowels
	'あ': "a", 'い': "i", 'う': "u", 'え': "e", 'お': "o",
	// K-row
	'か': "ka", 'き': "ki", 'く': "ku", 'け': "ke", 'こ': "ko",
	// S-row
	'さ': "sa", 'し': "shi", 'す': "su", 'せ': "se", 'そ': "so",
	// T-row
	'た': "ta", 'ち': "chi", 'つ': "tsu", 'て': "te", 'と': "to",
	// N-row
	'な': "na", 'に': "ni", 'ぬ': "nu", 'ね': "ne", 'の': "no",
	// H-row
	'は': "ha", 'ひ': "hi", 'ふ': "fu", 'へ': "he", 'ほ': "ho",
	// M-row
	'ま': "ma", 'み': "mi", 'む': "mu", 'め': "me", 'も': "mo",
	// Y-row
	'や': "ya", 'ゆ': "yu", 'よ': "yo",
	// R-row
	'ら': "ra", 'り': "ri", 'る': "ru", 'れ': "re", 'ろ': "ro",
	// W-row
	'わ': "wa", 'を': "wo",
	// N
	'ん': "n",
	// Voiced (dakuten) - G-row
	'が': "ga", 'ぎ': "gi", 'ぐ': "gu", 'げ': "ge", 'ご': "go",
	// Z-row
	'ざ': "za", 'じ': "ji", 'ず': "zu", 'ぜ': "ze", 'ぞ': "zo",
	// D-row
	'だ': "da", 'ぢ': "ji", 'づ': "zu", 'で': "de", 'ど': "do",
	// B-row
	'ば': "ba", 'び': "bi", 'ぶ': "bu", 'べ': "be", 'ぼ': "bo",
	// P-row (handakuten)
	'ぱ': "pa", 'ぴ': "pi", 'ぷ': "pu", 'ぺ': "pe", 'ぽ': "po",
	// Small characters
	'ゃ': "ya", 'ゅ': "yu", 'ょ': "yo",
	'ぁ': "a", 'ぃ': "i", 'ぅ': "u", 'ぇ': "e", 'ぉ': "o",
	'っ': "", // Small tsu - handled specially
	// Long vowel mark
	'ー': "",
}

// katakanaToRomaji maps katakana characters to romaji
var katakanaToRomaji = map[rune]string{
	// Basic vowels
	'ア': "a", 'イ': "i", 'ウ': "u", 'エ': "e", 'オ': "o",
	// K-row
	'カ': "ka", 'キ': "ki", 'ク': "ku", 'ケ': "ke", 'コ': "ko",
	// S-row
	'サ': "sa", 'シ': "shi", 'ス': "su", 'セ': "se", 'ソ': "so",
	// T-row
	'タ': "ta", 'チ': "chi", 'ツ': "tsu", 'テ': "te", 'ト': "to",
	// N-row
	'ナ': "na", 'ニ': "ni", 'ヌ': "nu", 'ネ': "ne", 'ノ': "no",
	// H-row
	'ハ': "ha", 'ヒ': "hi", 'フ': "fu", 'ヘ': "he", 'ホ': "ho",
	// M-row
	'マ': "ma", 'ミ': "mi", 'ム': "mu", 'メ': "me", 'モ': "mo",
	// Y-row
	'ヤ': "ya", 'ユ': "yu", 'ヨ': "yo",
	// R-row
	'ラ': "ra", 'リ': "ri", 'ル': "ru", 'レ': "re", 'ロ': "ro",
	// W-row
	'ワ': "wa", 'ヲ': "wo",
	// N
	'ン': "n",
	// Voiced (dakuten) - G-row
	'ガ': "ga", 'ギ': "gi", 'グ': "gu", 'ゲ': "ge", 'ゴ': "go",
	// Z-row
	'ザ': "za", 'ジ': "ji", 'ズ': "zu", 'ゼ': "ze", 'ゾ': "zo",
	// D-row
	'ダ': "da", 'ヂ': "ji", 'ヅ': "zu", 'デ': "de", 'ド': "do",
	// B-row
	'バ': "ba", 'ビ': "bi", 'ブ': "bu", 'ベ': "be", 'ボ': "bo",
	// P-row (handakuten)
	'パ': "pa", 'ピ': "pi", 'プ': "pu", 'ペ': "pe", 'ポ': "po",
	// Small characters
	'ャ': "ya", 'ュ': "yu", 'ョ': "yo",
	'ァ': "a", 'ィ': "i", 'ゥ': "u", 'ェ': "e", 'ォ': "o",
	'ッ': "", // Small tsu - handled specially
	// Extended katakana
	'ヴ': "vu",
	// Long vowel mark
	'ー': "",
}

// Extended katakana combinations (multi-character)
var katakanaExtended = map[string]string{
	"ファ": "fa", "フィ": "fi", "フェ": "fe", "フォ": "fo",
}

// Combination mappings for small ya/yu/yo
var hiraganaCombo = map[string]string{
	"きゃ": "kya", "きゅ": "kyu", "きょ": "kyo",
	"しゃ": "sha", "しゅ": "shu", "しょ": "sho",
	"ちゃ": "cha", "ちゅ": "chu", "ちょ": "cho",
	"にゃ": "nya", "にゅ": "nyu", "にょ": "nyo",
	"ひゃ": "hya", "ひゅ": "hyu", "ひょ": "hyo",
	"みゃ": "mya", "みゅ": "myu", "みょ": "myo",
	"りゃ": "rya", "りゅ": "ryu", "りょ": "ryo",
	"ぎゃ": "gya", "ぎゅ": "gyu", "ぎょ": "gyo",
	"じゃ": "ja", "じゅ": "ju", "じょ": "jo",
	"びゃ": "bya", "びゅ": "byu", "びょ": "byo",
	"ぴゃ": "pya", "ぴゅ": "pyu", "ぴょ": "pyo",
}

var katakanaCombo = map[string]string{
	"キャ": "kya", "キュ": "kyu", "キョ": "kyo",
	"シャ": "sha", "シュ": "shu", "ショ": "sho",
	"チャ": "cha", "チュ": "chu", "チョ": "cho",
	"ニャ": "nya", "ニュ": "nyu", "ニョ": "nyo",
	"ヒャ": "hya", "ヒュ": "hyu", "ヒョ": "hyo",
	"ミャ": "mya", "ミュ": "myu", "ミョ": "myo",
	"リャ": "rya", "リュ": "ryu", "リョ": "ryo",
	"ギャ": "gya", "ギュ": "gyu", "ギョ": "gyo",
	"ジャ": "ja", "ジュ": "ju", "ジョ": "jo",
	"ビャ": "bya", "ビュ": "byu", "ビョ": "byo",
	"ピャ": "pya", "ピュ": "pyu", "ピョ": "pyo",
	// Extended katakana combinations
	"ティ": "ti", "ディ": "di",
	"トゥ": "tu", "ドゥ": "du",
	"ファ": "fa", "フィ": "fi", "フェ": "fe", "フォ": "fo",
	"ウィ": "wi", "ウェ": "we", "ウォ": "wo",
	"ヴァ": "va", "ヴィ": "vi", "ヴェ": "ve", "ヴォ": "vo",
}

// ContainsJapanese checks if a string contains Japanese characters (Hiragana, Katakana, or Kanji)
func ContainsJapanese(s string) bool {
	for _, r := range s {
		if isHiragana(r) || isKatakana(r) || isKanji(r) {
			return true
		}
	}
	return false
}

// ContainsKana checks if a string contains Hiragana or Katakana (convertible to romaji)
func ContainsKana(s string) bool {
	for _, r := range s {
		if isHiragana(r) || isKatakana(r) {
			return true
		}
	}
	return false
}

func isHiragana(r rune) bool {
	return r >= hiraganaStart && r <= hiraganaEnd
}

func isKatakana(r rune) bool {
	return r >= katakanaStart && r <= katakanaEnd
}

func isKanji(r rune) bool {
	return r >= kanjiStart && r <= kanjiEnd
}

// ToRomaji converts Japanese kana (Hiragana/Katakana) to romaji
// Kanji characters are preserved as-is since they require dictionary lookup
func ToRomaji(s string) string {
	if !ContainsKana(s) {
		return s
	}

	runes := []rune(s)
	var result strings.Builder
	result.Grow(len(s) * 2) // Romaji is typically longer

	i := 0
	for i < len(runes) {
		r := runes[i]

		// Check for two-character combinations first
		if i+1 < len(runes) {
			combo := string(runes[i : i+2])
			if romaji, ok := hiraganaCombo[combo]; ok {
				result.WriteString(romaji)
				i += 2
				continue
			}
			if romaji, ok := katakanaCombo[combo]; ok {
				result.WriteString(romaji)
				i += 2
				continue
			}
		}

		// Handle small tsu (っ/ッ) - doubles the next consonant
		if r == 'っ' || r == 'ッ' {
			if i+1 < len(runes) {
				nextRune := runes[i+1]
				var nextRomaji string
				if romaji, ok := hiraganaToRomaji[nextRune]; ok {
					nextRomaji = romaji
				} else if romaji, ok := katakanaToRomaji[nextRune]; ok {
					nextRomaji = romaji
				}
				if len(nextRomaji) > 0 {
					result.WriteByte(nextRomaji[0]) // Double the consonant
				}
			}
			i++
			continue
		}

		// Handle long vowel mark (ー)
		if r == 'ー' {
			// Extend the previous vowel
			resultStr := result.String()
			if len(resultStr) > 0 {
				lastChar := resultStr[len(resultStr)-1]
				if lastChar == 'a' || lastChar == 'i' || lastChar == 'u' || lastChar == 'e' || lastChar == 'o' {
					result.WriteByte(lastChar)
				}
			}
			i++
			continue
		}

		// Single character conversion
		if romaji, ok := hiraganaToRomaji[r]; ok {
			result.WriteString(romaji)
			i++
			continue
		}

		if romaji, ok := katakanaToRomaji[r]; ok {
			result.WriteString(romaji)
			i++
			continue
		}

		// Keep non-Japanese characters as-is
		if unicode.IsSpace(r) {
			result.WriteRune(' ')
		} else {
			result.WriteRune(r)
		}
		i++
	}

	return result.String()
}

// GetRomajiVariants returns search variants for Japanese text
// Returns the original string plus romaji version if applicable
func GetRomajiVariants(s string) []string {
	variants := []string{s}

	if ContainsKana(s) {
		romaji := ToRomaji(s)
		if romaji != s && strings.TrimSpace(romaji) != "" {
			variants = append(variants, romaji)
		}
	}

	return variants
}
