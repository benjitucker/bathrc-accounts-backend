package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"benjitucker/bathrc-accounts/db"
)

// Training prices come from the Training form's definition, so they stay editable in Jotform.
//
// For each venue the form has two price fields, found by their unique names:
//   <venue>mem     the member price
//   <venue>nonmem  the non-member price
// where <venue> is the venue's value in the form (e.g. "RiverMead" -> "rivermeadmem"), compared
// ignoring case. A new venue therefore needs no change here. The three original venues use
// shorter names, mapped in legacyPricePrefixes.

// legacyPricePrefixes maps venue values to the price field prefixes the form already used before
// the naming convention above.
var legacyPricePrefixes = map[string]string{
	"westwilts":    "wwec",
	"widbrook":     "wid",
	"stockleyfarm": "slf",
}

// priceValueKeys are the question properties that may hold a price field's value, in order.
var priceValueKeys = []string{"defaultValue", "value"}

const priceCacheDuration = 5 * time.Minute

type venuePrices struct {
	MemberPence    int64
	NonMemberPence int64
}

// formPriceFields holds the price fields read from the form definition.
type formPriceFields struct {
	// pence by lower-case unique name, e.g. "wwecmem" -> 2100
	pence map[string]int64
	// raw question JSON of every field whose name ends in "mem", for diagnostics
	raw map[string]json.RawMessage
}

// parsePriceFields extracts the price fields from a Jotform "form questions" response, which
// maps question ids to question objects.
func parsePriceFields(questionsJSON []byte) (*formPriceFields, error) {
	var questions map[string]json.RawMessage
	if err := json.Unmarshal(questionsJSON, &questions); err != nil {
		return nil, fmt.Errorf("failed to parse form questions: %w", err)
	}

	fields := &formPriceFields{pence: map[string]int64{}, raw: map[string]json.RawMessage{}}
	for _, raw := range questions {
		var properties map[string]any
		if err := json.Unmarshal(raw, &properties); err != nil {
			continue
		}
		name, _ := properties["name"].(string)
		name = strings.ToLower(strings.TrimSpace(name))
		if !strings.HasSuffix(name, "mem") {
			continue
		}
		fields.raw[name] = raw

		for _, key := range priceValueKeys {
			value, ok := properties[key]
			if !ok {
				continue
			}
			if pence, ok := parsePoundsToPence(fmt.Sprint(value)); ok {
				fields.pence[name] = pence
				break
			}
		}
	}
	return fields, nil
}

// parsePoundsToPence converts a price in pounds such as "21", "21.5", "£21.50" to pence.
func parsePoundsToPence(value string) (int64, bool) {
	value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "£"))
	if value == "" {
		return 0, false
	}
	pounds, err := strconv.ParseFloat(value, 64)
	if err != nil || pounds < 0 || math.IsNaN(pounds) || math.IsInf(pounds, 0) {
		return 0, false
	}
	return int64(math.Round(pounds * 100)), true
}

// pricesForVenue finds the member and non-member prices for a venue value.
func (f *formPriceFields) pricesForVenue(venue string) (venuePrices, bool) {
	key := strings.ToLower(strings.TrimSpace(venue))
	if key == "" {
		return venuePrices{}, false
	}
	prefixes := []string{key}
	if legacy, ok := legacyPricePrefixes[key]; ok {
		prefixes = append(prefixes, legacy)
	}
	for _, prefix := range prefixes {
		member, okMember := f.pence[prefix+"mem"]
		nonMember, okNonMember := f.pence[prefix+"nonmem"]
		if okMember && okNonMember {
			return venuePrices{MemberPence: member, NonMemberPence: nonMember}, true
		}
	}
	return venuePrices{}, false
}

// diagnostic describes the price fields that were found, for the "prices not found" email.
func (f *formPriceFields) diagnostic() string {
	if len(f.raw) == 0 {
		return "No fields with a unique name ending in \"mem\" were found in the form."
	}
	names := make([]string, 0, len(f.raw))
	for name := range f.raw {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("Price fields found in the form (unique name: pence, then the raw question definition):\n")
	for _, name := range names {
		pence, ok := f.pence[name]
		value := "no price read"
		if ok {
			value = fmt.Sprintf("%d", pence)
		}
		raw := string(f.raw[name])
		if len(raw) > 1500 {
			raw = raw[:1500] + "..."
		}
		fmt.Fprintf(&b, "\n%s: %s\n%s\n", name, value, raw)
	}
	return b.String()
}

var priceCache struct {
	sync.Mutex
	fields  *formPriceFields
	expires time.Time
}

// loadPriceFields reads the price fields from the Training form, caching them briefly.
func loadPriceFields() (*formPriceFields, error) {
	priceCache.Lock()
	defer priceCache.Unlock()

	if priceCache.fields != nil && time.Now().Before(priceCache.expires) {
		return priceCache.fields, nil
	}

	data, err := jotformClient.GetFormQuestions(trainingFormID)
	if err != nil {
		return nil, fmt.Errorf("failed to get form questions: %w", err)
	}
	fields, err := parsePriceFields(data)
	if err != nil {
		return nil, err
	}

	priceCache.fields = fields
	priceCache.expires = time.Now().Add(priceCacheDuration)
	return fields, nil
}

// chargeAmountPence is the amount to charge for a booking: the member price when the
// membership is valid on the session date, otherwise the non-member price.
func chargeAmountPence(booking *db.TrainingSubmission) int64 {
	if booking.ActualCurrMem {
		return booking.MemberPricePence
	}
	return booking.NonMemberPricePence
}
