package housing_list

import (
	"errors"
	"math/rand"
	"sync"
)

var (
	list []HousingLocation
	mtx  sync.RWMutex
	once sync.Once
)

func init() {
	once.Do(initialiseList)
}

func initialiseList() {
	list = []HousingLocation{
		{
			ID:             0,
			Name:           "Acme Fresh Start Housing",
			City:           "Chicago",
			State:          "IL",
			Photo:          "https://angular.io/assets/images/tutorials/faa/bernard-hermant-CLKGGwIBTaY-unsplash.jpg",
			AvailableUnits: 4,
			Wifi:           true,
			Laundry:        true,
		},
		{
			ID:             1,
			Name:           "A113 Transitional Housing",
			City:           "Santa Monica",
			State:          "CA",
			Photo:          "https://angular.io/assets/images/tutorials/faa/brandon-griggs-wR11KBaB86U-unsplash.jpg",
			AvailableUnits: 0,
			Wifi:           false,
			Laundry:        true,
		},
		{
			ID:             2,
			Name:           "Warm Beds Housing Support",
			City:           "Juneau",
			State:          "AK",
			Photo:          "https://angular.io/assets/images/tutorials/faa/i-do-nothing-but-love-lAyXdl1-Wmc-unsplash.jpg",
			AvailableUnits: 1,
			Wifi:           false,
			Laundry:        false,
		},
		{
			ID:             3,
			Name:           "Homesteady Housing",
			City:           "Chicago",
			State:          "IL",
			Photo:          "https://angular.io/assets/images/tutorials/faa/ian-macdonald-W8z6aiwfi1E-unsplash.jpg",
			AvailableUnits: 1,
			Wifi:           true,
			Laundry:        false,
		},
		{
			ID:             4,
			Name:           "Happy Homes Group",
			City:           "Gary",
			State:          "IN",
			Photo:          "https://angular.io/assets/images/tutorials/faa/krzysztof-hepner-978RAXoXnH4-unsplash.jpg",
			AvailableUnits: 1,
			Wifi:           true,
			Laundry:        false,
		},
		{
			ID:             5,
			Name:           "Hopeful Apartment Group",
			City:           "Oakland",
			State:          "CA",
			Photo:          "https://angular.io/assets/images/tutorials/faa/r-architecture-JvQ0Q5IkeMM-unsplash.jpg",
			AvailableUnits: 2,
			Wifi:           true,
			Laundry:        true,
		},
		{
			ID:             6,
			Name:           "Seriously Safe Towns",
			City:           "Oakland",
			State:          "CA",
			Photo:          "https://angular.io/assets/images/tutorials/faa/phil-hearing-IYfp2Ixe9nM-unsplash.jpg",
			AvailableUnits: 5,
			Wifi:           true,
			Laundry:        true,
		},
		{
			ID:             7,
			Name:           "Hopeful Housing Solutions",
			City:           "Oakland",
			State:          "CA",
			Photo:          "https://angular.io/assets/images/tutorials/faa/r-architecture-GGupkreKwxA-unsplash.jpg",
			AvailableUnits: 2,
			Wifi:           true,
			Laundry:        true,
		},
		{
			ID:             8,
			Name:           "Seriously Safe Towns",
			City:           "Oakland",
			State:          "CA",
			Photo:          "https://angular.io/assets/images/tutorials/faa/saru-robert-9rP3mxf8qWI-unsplash.jpg",
			AvailableUnits: 10,
			Wifi:           false,
			Laundry:        false,
		},
		{
			ID:             9,
			Name:           "Capital Safe Towns",
			City:           "Portland",
			State:          "OR",
			Photo:          "https://angular.io/assets/images/tutorials/faa/webaliser-_TPTXZd9mOo-unsplash.jpg",
			AvailableUnits: 6,
			Wifi:           true,
			Laundry:        true,
		},
	}
}

// To-do data structure for task with a description of what to do
type HousingLocation struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	City           string `json:"city"`
	State          string `json:"state"`
	Photo          string `json:"photo"`
	AvailableUnits int    `json:"availableUnits"`
	Wifi           bool   `json:"wifi"`
	Laundry        bool   `json:"laundry"`
}

// Get retrieves all elements from the todo list
func Get() []HousingLocation {
	return list
}

func GetHousingLocationById(id int) (*HousingLocation, error) {
	mtx.RLock()
	defer mtx.RUnlock()
	for _, t := range list {
		if isMatchingID(t.ID, id) {
			t.AvailableUnits = rand.Int()
			return &t, nil
		}
	}
	return nil, errors.New("could not find todo based on id")
}

func isMatchingID(a int, b int) bool {
	return a == b
}

/*
// Add will add a new todo based on a message
func Add(message string) string {
	t := newHousingLocation(message)
	mtx.Lock()
	list = append(list, t)
	mtx.Unlock()
	return t.ID
}

// Delete will remove a HousingLocation from the Todo list
func Delete(id string) error {
	location, err := findHousingLocationLocation(id)
	if err != nil {
		return err
	}
	removeElementByLocation(location)
	return nil
}

// Complete will set the complete boolean to true, marking a todo as
// completed
func Complete(id string) error {
	location, err := findHousingLocationLocation(id)
	if err != nil {
		return err
	}
	setHousingLocationCompleteByLocation(location)
	return nil
}

func newHousingLocation(msg string) Todo {
	return HousingLocation{
		ID:       xid.New().String(),
		Message:  msg,
		Complete: false,
	}
}

func removeElementByLocation(i int) {
	mtx.Lock()
	list = append(list[:i], list[i+1:]...)
	mtx.Unlock()
}

func setHousingLocationCompleteByLocation(location int) {
	mtx.Lock()
	list[location].Complete = true
	mtx.Unlock()
}

*/
