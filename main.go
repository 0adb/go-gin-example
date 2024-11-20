package main

import (
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Item struct {
	ShortDescription string `json:"shortDescription" validate:"required"`
	Price            string `json:"price" validate:"required"`
}

type Receipt struct {
	Retailer     string `json:"retailer" validate:"required`
	PurchaseDate string `json:"purchaseDate" validate:"required"`
	PurchaseTime string `json:"purchaseTime" validate:"required"`
	Items        []Item `json:"items" validate:"required"`
	Total        string `json:"total" validate:"required"`
}

var (
	receipts map[uuid.UUID]Receipt = make(map[uuid.UUID]Receipt)
	points map[uuid.UUID]int64 = make(map[uuid.UUID]int64)
	muReceipts sync.Mutex
	muPoints sync.Mutex

	retailer_re = regexp.MustCompile("^[\\w\\s\\-&]+$")
	price_re = regexp.MustCompile("^\\d+\\.\\d{2}$")
	shortdesc_re = regexp.MustCompile("^[\\w\\s\\-]+$")
	alphanum_re = regexp.MustCompile(`[a-zA-Z0-9]`)
)	

func ReceiptValidation(sl validator.StructLevel) {
	receipt := sl.Current().Interface().(Receipt)
	{
		if !(retailer_re.MatchString(receipt.Retailer)) {
			sl.ReportError(receipt.Retailer,
				"Retailer",
				"retailer",
				"retailerformat",
				"")
		}
	}

	{
		_, err := time.Parse("2006-01-02", receipt.PurchaseDate)
		if err != nil {
			sl.ReportError(receipt.PurchaseDate,
				"PurchaseDate",
				"purchaseDate",
				"purchasedateformat",
				"")
		}
	}

	{
		_, err := time.Parse("15:04", receipt.PurchaseTime)
		if err != nil {
			sl.ReportError(receipt.PurchaseTime,
				"PurchaseTime",
				"purchaseTime",
				"purchasetimeformat",
				"")
		}
	}

	if !(price_re.MatchString(receipt.Total)) {
		sl.ReportError(receipt.Total, "Total", "total", "totalformat", "")
		return
	}

	total, err := strconv.ParseFloat(receipt.Total, 64)

	if err != nil {
		sl.ReportError(receipt.Total, "Total", "total", "totalnumber", "")
		return
	}

	if len(receipt.Items) < 1 {
		sl.ReportError(receipt.Items, "Items", "items", "emptyitems", "")
		return
	}

	var price_sum float64 = 0

	for index, item := range receipt.Items {
		if !(price_re.MatchString(item.Price)) {
			sl.ReportError(item.Price,
				fmt.Sprintf("Items[%d].Price", index),
				fmt.Sprintf("items[%d].price", index),
				"itempriceformat",
				"")
			return
		}

		if !(shortdesc_re.MatchString(item.ShortDescription)) {
			sl.ReportError(item.ShortDescription,
				fmt.Sprintf("Items[%d].ShortDescription", index),
				fmt.Sprintf("items[%d].shortDescription", index),
				"itemdescformat",
				"")
			return
		}

		item_price, err := strconv.ParseFloat(item.Price, 64)
		if err != nil {
			sl.ReportError(item.Price,
				fmt.Sprintf("Items[%d].Price", index),
				fmt.Sprintf("items[%d].price", index),
				"itempricenumber",
				"")
			return
		}
		price_sum += item_price
	}

	difference := price_sum - total

	if (difference <= -0.01) || (difference >= 0.01) {
		sl.ReportError(receipt.Total,
			"Total",
			"total",
			"totalmatchsumprice",
			"")
	}
}

func main() {
	router := gin.Default()

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterStructValidation(ReceiptValidation, Receipt{})
	}

	router.POST("/receipts/process", postReceipt)
	router.GET("/receipts/:id/points", getPoints)
	router.Run(":8080")
}

func postReceipt(c *gin.Context) {
	var newReceipt Receipt
	if err := c.BindJSON(&newReceipt); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	var id uuid.UUID = uuid.New()
	muReceipts.Lock()
	receipts[id] = newReceipt
	muReceipts.Unlock()

	c.JSON(http.StatusOK, gin.H{"id": id.String()})
}

func calculatePoints(r Receipt) int64 {
	var total int64 = 0
	{ // One point for every alphanumeric character in the retailer name.
		matches := alphanum_re.FindAllString(r.Retailer, -1)
		total += int64(len(matches))
	}
	{ // 50 points if the total is a round dollar amount with no cents.
		// 25 points if the total is a multiple of 0.25.
		s := r.Total
		var cents string = s[len(s)-2:]
		if cents == "00" {
			total += 50
		}
		multiples := map[string]struct{}{
			"50": {},
			"25": {},
			"75": {},
			"00": {},
		}
		if _, exists := multiples[cents]; exists {
			total += 25
		}
	}
	{ // 5 points for every two items on the receipt.
		total += int64((len(r.Items) >> 1) * 5)
	}
	{ // If the trimmed length of the item description is a multiple of 3,
		// multiply the price by 0.2 and round up to the nearest integer.
		// The result is the number of points earned.
		for _, item := range r.Items {
			item_trimmed_desc := strings.TrimSpace(item.ShortDescription)
			if len(item_trimmed_desc)%3 == 0 {
				// Assuming no errors parsing this because this passed validation.
				item_price, _ := strconv.ParseFloat(item.Price, 64)
				item_price *= 0.2
				total += int64(math.Ceil(item_price))
			}
		}
	}
	{ //  6 points if the day in the purchase date is odd
		odds := map[byte]struct{}{
			'1': {},
			'3': {},
			'5': {},
			'7': {},
			'9': {},
		}
		s := r.PurchaseDate
		dateLastDigit := s[len(s)-1]
		if _, exists := odds[dateLastDigit]; exists {
			total += 6
		}
	}
	{ // 10 points if the time of purchase is after 2:00pm and before 4:00pm.
		s := r.PurchaseTime
		if strings.Compare("14:00", s) == -1 && strings.Compare(s, "16:00") == -1 {
			total += 10
		}
	}
	return total

}

func getPoints(c *gin.Context) {
	id := c.Param("id")
	uuidParsed, err := uuid.Parse(id)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
	}

	var exists bool
	var entry int64
	muPoints.Lock()
	entry, exists = points[uuidParsed]
	muPoints.Unlock()

	if exists {
		c.JSON(http.StatusOK, gin.H{"points": entry})
		return
	}

	var receipt Receipt
	muReceipts.Lock()
	receipt, exists = receipts[uuidParsed]
	muReceipts.Unlock()

	if !exists {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	pointCount := calculatePoints(receipt)
	muPoints.Lock()
	points[uuidParsed] = pointCount
	muPoints.Unlock()
	c.JSON(http.StatusOK, gin.H{"points": pointCount})

}
