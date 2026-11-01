package main

import "mongoscale/internal/models"

type categorySpec struct {
	name     string
	nouns    []string
	minPrice float64
	maxPrice float64
}

var categories = []categorySpec{
	{"Electronics", []string{"Headphones", "Speaker", "Smartwatch", "Power Bank", "Camera", "Tablet"}, 15, 900},
	{"Home & Kitchen", []string{"Blender", "Air Fryer", "Kettle", "Cookware Set", "Vacuum", "Lamp"}, 10, 400},
	{"Fashion", []string{"Jacket", "Sneakers", "Backpack", "Sunglasses", "Watch", "Hoodie"}, 12, 350},
	{"Sports & Outdoors", []string{"Yoga Mat", "Tent", "Dumbbell Set", "Bicycle Helmet", "Water Bottle", "Trekking Pole"}, 8, 500},
	{"Books & Stationery", []string{"Notebook", "Planner", "Pen Set", "Desk Organizer", "Sketchbook", "Reading Light"}, 3, 80},
	{"Beauty & Care", []string{"Face Serum", "Trimmer", "Hair Dryer", "Perfume", "Skincare Kit", "Electric Toothbrush"}, 6, 220},
	{"Toys & Games", []string{"Board Game", "Puzzle", "Building Set", "Remote Car", "Plush Toy", "Card Game"}, 5, 150},
	{"Office", []string{"Monitor Stand", "Ergonomic Chair", "Keyboard", "Mouse", "Webcam", "Docking Station"}, 12, 650},
	{"Grocery & Gourmet", []string{"Coffee Beans", "Olive Oil", "Tea Collection", "Protein Bars", "Spice Box", "Dark Chocolate"}, 2, 60},
	{"Garden", []string{"Planter", "Pruning Shears", "Sprinkler", "Garden Lights", "Tool Kit", "Seed Collection"}, 4, 200},
}

var adjectives = []string{"Pro", "Smart", "Classic", "Ultra", "Eco", "Premium", "Compact", "Wireless", "Essential", "Deluxe", "Portable", "Heritage"}

var firstNames = []string{"Aarav", "Vivaan", "Aditya", "Diya", "Ananya", "Ishaan", "Kavya", "Rohan", "Meera", "Arjun", "Saanvi", "Kabir", "Priya", "Neha", "Rahul", "Sneha", "Vikram", "Pooja", "Aman", "Ritika"}

var lastNames = []string{"Sharma", "Verma", "Gupta", "Singh", "Patel", "Reddy", "Nair", "Iyer", "Mehta", "Joshi", "Kapoor", "Chopra", "Das", "Bose", "Malhotra", "Tyagi", "Yadav", "Mishra", "Pandey", "Khan"}

var cities = []struct {
	city  string
	state string
	zip   string
}{
	{"Bengaluru", "Karnataka", "560001"},
	{"Mumbai", "Maharashtra", "400001"},
	{"Delhi", "Delhi", "110001"},
	{"Hyderabad", "Telangana", "500001"},
	{"Chennai", "Tamil Nadu", "600001"},
	{"Pune", "Maharashtra", "411001"},
	{"Kolkata", "West Bengal", "700001"},
	{"Ghaziabad", "Uttar Pradesh", "201001"},
	{"Jaipur", "Rajasthan", "302001"},
	{"Ahmedabad", "Gujarat", "380001"},
}

var reviewComments = []string{
	"Exactly what I was looking for, great quality.",
	"Delivery was fast and the packaging was solid.",
	"Good value for money, would recommend.",
	"Works well but the instructions could be clearer.",
	"Stopped working after a few weeks, disappointed.",
	"Looks even better in person.",
	"Perfect gift, the recipient loved it.",
	"Average product, nothing special.",
	"Excellent build quality and very easy to use.",
	"Bought a second one for the office.",
}

type orderStatusWeight struct {
	status  models.OrderStatus
	payment models.PaymentStatus
	weight  int
}

var orderStatusMix = []orderStatusWeight{
	{models.OrderDelivered, models.PaymentPaid, 45},
	{models.OrderShipped, models.PaymentPaid, 15},
	{models.OrderPaid, models.PaymentPaid, 15},
	{models.OrderPending, models.PaymentPending, 15},
	{models.OrderCancelled, models.PaymentRefunded, 10},
}

var ratingWeights = []int{5, 7, 13, 35, 40}
