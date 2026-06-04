package services

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// SeedLocations populates the country/state/city/postcode reference tables
// with India-only data. Idempotent: every insert is `ON CONFLICT DO NOTHING`,
// so a re-run is a no-op once the data is in place.
//
// Scope today: India + 36 states/UTs + ~50 major cities + a representative
// set of ~250 PIN codes drawn from those cities. The schema is
// country-agnostic so a second country can be added by extending the
// `seedCountries`, `seedStates`, ... slices below.
//
// Sourced from mark8ly's `services/platform-api/seed/locations.json` for
// the country + state rows. City + postcode data is curated for the
// initial release; the full Indian PIN dataset (~150k entries) can be
// loaded from a follow-on seed file once the data-refresh story is
// agreed.
func SeedLocations(ctx context.Context, db *gorm.DB) error {
	doNothing := clause.OnConflict{DoNothing: true}

	if err := db.WithContext(ctx).
		Clauses(doNothing).
		CreateInBatches(seedCountries, 100).Error; err != nil {
		return fmt.Errorf("seed countries: %w", err)
	}
	if err := db.WithContext(ctx).
		Clauses(doNothing).
		CreateInBatches(seedStates, 100).Error; err != nil {
		return fmt.Errorf("seed states: %w", err)
	}
	if err := db.WithContext(ctx).
		Clauses(doNothing).
		CreateInBatches(seedCities, 100).Error; err != nil {
		return fmt.Errorf("seed cities: %w", err)
	}
	if err := db.WithContext(ctx).
		Clauses(doNothing).
		CreateInBatches(seedPostcodes, 200).Error; err != nil {
		return fmt.Errorf("seed postcodes: %w", err)
	}
	return nil
}

// seedCountries — India only. The schema accepts more; the seed is
// deliberately scoped to the country we serve today.
var seedCountries = []models.Country{
	{Code: "IN", Name: "India", NativeName: "भारत", CallingCode: "+91", CurrencyCode: "INR", FlagEmoji: "🇮🇳", Region: "Asia"},
}

// seedStates — all 36 Indian states + Union Territories.
// State codes follow ISO 3166-2:IN. Lifted verbatim from
// mark8ly/services/platform-api/seed/locations.json.
var seedStates = []models.State{
	{ID: "IN-AN", CountryCode: "IN", Code: "AN", Name: "Andaman and Nicobar Islands", Type: "territory"},
	{ID: "IN-AP", CountryCode: "IN", Code: "AP", Name: "Andhra Pradesh", Type: "state"},
	{ID: "IN-AR", CountryCode: "IN", Code: "AR", Name: "Arunachal Pradesh", Type: "state"},
	{ID: "IN-AS", CountryCode: "IN", Code: "AS", Name: "Assam", Type: "state"},
	{ID: "IN-BR", CountryCode: "IN", Code: "BR", Name: "Bihar", Type: "state"},
	{ID: "IN-CH", CountryCode: "IN", Code: "CH", Name: "Chandigarh", Type: "territory"},
	{ID: "IN-CT", CountryCode: "IN", Code: "CT", Name: "Chhattisgarh", Type: "state"},
	{ID: "IN-DH", CountryCode: "IN", Code: "DH", Name: "Dadra and Nagar Haveli and Daman and Diu", Type: "territory"},
	{ID: "IN-DL", CountryCode: "IN", Code: "DL", Name: "Delhi", Type: "territory"},
	{ID: "IN-GA", CountryCode: "IN", Code: "GA", Name: "Goa", Type: "state"},
	{ID: "IN-GJ", CountryCode: "IN", Code: "GJ", Name: "Gujarat", Type: "state"},
	{ID: "IN-HR", CountryCode: "IN", Code: "HR", Name: "Haryana", Type: "state"},
	{ID: "IN-HP", CountryCode: "IN", Code: "HP", Name: "Himachal Pradesh", Type: "state"},
	{ID: "IN-JK", CountryCode: "IN", Code: "JK", Name: "Jammu and Kashmir", Type: "territory"},
	{ID: "IN-JH", CountryCode: "IN", Code: "JH", Name: "Jharkhand", Type: "state"},
	{ID: "IN-KA", CountryCode: "IN", Code: "KA", Name: "Karnataka", Type: "state"},
	{ID: "IN-KL", CountryCode: "IN", Code: "KL", Name: "Kerala", Type: "state"},
	{ID: "IN-LA", CountryCode: "IN", Code: "LA", Name: "Ladakh", Type: "territory"},
	{ID: "IN-LD", CountryCode: "IN", Code: "LD", Name: "Lakshadweep", Type: "territory"},
	{ID: "IN-MP", CountryCode: "IN", Code: "MP", Name: "Madhya Pradesh", Type: "state"},
	{ID: "IN-MH", CountryCode: "IN", Code: "MH", Name: "Maharashtra", Type: "state"},
	{ID: "IN-MN", CountryCode: "IN", Code: "MN", Name: "Manipur", Type: "state"},
	{ID: "IN-ML", CountryCode: "IN", Code: "ML", Name: "Meghalaya", Type: "state"},
	{ID: "IN-MZ", CountryCode: "IN", Code: "MZ", Name: "Mizoram", Type: "state"},
	{ID: "IN-NL", CountryCode: "IN", Code: "NL", Name: "Nagaland", Type: "state"},
	{ID: "IN-OR", CountryCode: "IN", Code: "OR", Name: "Odisha", Type: "state"},
	{ID: "IN-PY", CountryCode: "IN", Code: "PY", Name: "Puducherry", Type: "territory"},
	{ID: "IN-PB", CountryCode: "IN", Code: "PB", Name: "Punjab", Type: "state"},
	{ID: "IN-RJ", CountryCode: "IN", Code: "RJ", Name: "Rajasthan", Type: "state"},
	{ID: "IN-SK", CountryCode: "IN", Code: "SK", Name: "Sikkim", Type: "state"},
	{ID: "IN-TN", CountryCode: "IN", Code: "TN", Name: "Tamil Nadu", Type: "state"},
	{ID: "IN-TG", CountryCode: "IN", Code: "TG", Name: "Telangana", Type: "state"},
	{ID: "IN-TR", CountryCode: "IN", Code: "TR", Name: "Tripura", Type: "state"},
	{ID: "IN-UP", CountryCode: "IN", Code: "UP", Name: "Uttar Pradesh", Type: "state"},
	{ID: "IN-UK", CountryCode: "IN", Code: "UK", Name: "Uttarakhand", Type: "state"},
	{ID: "IN-WB", CountryCode: "IN", Code: "WB", Name: "West Bengal", Type: "state"},
}

// seedCities — major Indian metros and tier-2 cities. The ID is
// deterministic so re-seeds collapse to a no-op.
var seedCities = []models.City{
	// Karnataka
	{ID: "IN-KA-bengaluru", StateID: "IN-KA", Name: "Bengaluru", IsMajor: true, Latitude: 12.9716, Longitude: 77.5946},
	{ID: "IN-KA-mysuru", StateID: "IN-KA", Name: "Mysuru", Latitude: 12.2958, Longitude: 76.6394},
	{ID: "IN-KA-mangaluru", StateID: "IN-KA", Name: "Mangaluru", Latitude: 12.9141, Longitude: 74.8560},
	{ID: "IN-KA-hubli", StateID: "IN-KA", Name: "Hubballi", Latitude: 15.3647, Longitude: 75.1240},

	// Maharashtra
	{ID: "IN-MH-mumbai", StateID: "IN-MH", Name: "Mumbai", IsMajor: true, Latitude: 19.0760, Longitude: 72.8777},
	{ID: "IN-MH-pune", StateID: "IN-MH", Name: "Pune", IsMajor: true, Latitude: 18.5204, Longitude: 73.8567},
	{ID: "IN-MH-nagpur", StateID: "IN-MH", Name: "Nagpur", Latitude: 21.1458, Longitude: 79.0882},
	{ID: "IN-MH-nashik", StateID: "IN-MH", Name: "Nashik", Latitude: 19.9975, Longitude: 73.7898},
	{ID: "IN-MH-thane", StateID: "IN-MH", Name: "Thane", Latitude: 19.2183, Longitude: 72.9781},

	// Delhi
	{ID: "IN-DL-new-delhi", StateID: "IN-DL", Name: "New Delhi", IsMajor: true, Latitude: 28.6139, Longitude: 77.2090},

	// Tamil Nadu
	{ID: "IN-TN-chennai", StateID: "IN-TN", Name: "Chennai", IsMajor: true, Latitude: 13.0827, Longitude: 80.2707},
	{ID: "IN-TN-coimbatore", StateID: "IN-TN", Name: "Coimbatore", Latitude: 11.0168, Longitude: 76.9558},
	{ID: "IN-TN-madurai", StateID: "IN-TN", Name: "Madurai", Latitude: 9.9252, Longitude: 78.1198},

	// Telangana / AP
	{ID: "IN-TG-hyderabad", StateID: "IN-TG", Name: "Hyderabad", IsMajor: true, Latitude: 17.3850, Longitude: 78.4867},
	{ID: "IN-AP-visakhapatnam", StateID: "IN-AP", Name: "Visakhapatnam", Latitude: 17.6868, Longitude: 83.2185},
	{ID: "IN-AP-vijayawada", StateID: "IN-AP", Name: "Vijayawada", Latitude: 16.5062, Longitude: 80.6480},

	// West Bengal
	{ID: "IN-WB-kolkata", StateID: "IN-WB", Name: "Kolkata", IsMajor: true, Latitude: 22.5726, Longitude: 88.3639},
	{ID: "IN-WB-howrah", StateID: "IN-WB", Name: "Howrah", Latitude: 22.5958, Longitude: 88.2636},

	// Gujarat
	{ID: "IN-GJ-ahmedabad", StateID: "IN-GJ", Name: "Ahmedabad", IsMajor: true, Latitude: 23.0225, Longitude: 72.5714},
	{ID: "IN-GJ-surat", StateID: "IN-GJ", Name: "Surat", Latitude: 21.1702, Longitude: 72.8311},
	{ID: "IN-GJ-vadodara", StateID: "IN-GJ", Name: "Vadodara", Latitude: 22.3072, Longitude: 73.1812},
	{ID: "IN-GJ-rajkot", StateID: "IN-GJ", Name: "Rajkot", Latitude: 22.3039, Longitude: 70.8022},

	// Rajasthan
	{ID: "IN-RJ-jaipur", StateID: "IN-RJ", Name: "Jaipur", IsMajor: true, Latitude: 26.9124, Longitude: 75.7873},
	{ID: "IN-RJ-jodhpur", StateID: "IN-RJ", Name: "Jodhpur", Latitude: 26.2389, Longitude: 73.0243},
	{ID: "IN-RJ-udaipur", StateID: "IN-RJ", Name: "Udaipur", Latitude: 24.5854, Longitude: 73.7125},

	// Uttar Pradesh
	{ID: "IN-UP-lucknow", StateID: "IN-UP", Name: "Lucknow", IsMajor: true, Latitude: 26.8467, Longitude: 80.9462},
	{ID: "IN-UP-kanpur", StateID: "IN-UP", Name: "Kanpur", Latitude: 26.4499, Longitude: 80.3319},
	{ID: "IN-UP-varanasi", StateID: "IN-UP", Name: "Varanasi", Latitude: 25.3176, Longitude: 82.9739},
	{ID: "IN-UP-agra", StateID: "IN-UP", Name: "Agra", Latitude: 27.1767, Longitude: 78.0081},
	{ID: "IN-UP-noida", StateID: "IN-UP", Name: "Noida", Latitude: 28.5355, Longitude: 77.3910},

	// Haryana
	{ID: "IN-HR-gurugram", StateID: "IN-HR", Name: "Gurugram", IsMajor: true, Latitude: 28.4595, Longitude: 77.0266},
	{ID: "IN-HR-faridabad", StateID: "IN-HR", Name: "Faridabad", Latitude: 28.4089, Longitude: 77.3178},

	// MP
	{ID: "IN-MP-bhopal", StateID: "IN-MP", Name: "Bhopal", Latitude: 23.2599, Longitude: 77.4126},
	{ID: "IN-MP-indore", StateID: "IN-MP", Name: "Indore", Latitude: 22.7196, Longitude: 75.8577},

	// Kerala
	{ID: "IN-KL-thiruvananthapuram", StateID: "IN-KL", Name: "Thiruvananthapuram", Latitude: 8.5241, Longitude: 76.9366},
	{ID: "IN-KL-kochi", StateID: "IN-KL", Name: "Kochi", Latitude: 9.9312, Longitude: 76.2673},
	{ID: "IN-KL-kozhikode", StateID: "IN-KL", Name: "Kozhikode", Latitude: 11.2588, Longitude: 75.7804},

	// Punjab
	{ID: "IN-PB-amritsar", StateID: "IN-PB", Name: "Amritsar", Latitude: 31.6340, Longitude: 74.8723},
	{ID: "IN-PB-ludhiana", StateID: "IN-PB", Name: "Ludhiana", Latitude: 30.9010, Longitude: 75.8573},

	// Chandigarh
	{ID: "IN-CH-chandigarh", StateID: "IN-CH", Name: "Chandigarh", Latitude: 30.7333, Longitude: 76.7794},

	// Bihar
	{ID: "IN-BR-patna", StateID: "IN-BR", Name: "Patna", Latitude: 25.5941, Longitude: 85.1376},

	// Odisha
	{ID: "IN-OR-bhubaneswar", StateID: "IN-OR", Name: "Bhubaneswar", Latitude: 20.2961, Longitude: 85.8245},

	// Assam
	{ID: "IN-AS-guwahati", StateID: "IN-AS", Name: "Guwahati", Latitude: 26.1445, Longitude: 91.7362},

	// Uttarakhand
	{ID: "IN-UK-dehradun", StateID: "IN-UK", Name: "Dehradun", Latitude: 30.3165, Longitude: 78.0322},

	// Chhattisgarh
	{ID: "IN-CT-raipur", StateID: "IN-CT", Name: "Raipur", Latitude: 21.2514, Longitude: 81.6296},

	// Jharkhand
	{ID: "IN-JH-ranchi", StateID: "IN-JH", Name: "Ranchi", Latitude: 23.3441, Longitude: 85.3096},

	// Goa
	{ID: "IN-GA-panaji", StateID: "IN-GA", Name: "Panaji", Latitude: 15.4909, Longitude: 73.8278},

	// HP
	{ID: "IN-HP-shimla", StateID: "IN-HP", Name: "Shimla", Latitude: 31.1048, Longitude: 77.1734},
}

// seedPostcodes — representative PIN codes for the seeded cities.
// Selection prioritizes recognizable neighborhoods so the autocomplete
// can be smoke-tested by typing a familiar area name. The dataset
// intentionally doesn't try to be exhaustive — the full ~150k Indian
// PIN dataset can be loaded in a follow-up via the same seeder shape.
var seedPostcodes = []models.Postcode{
	// Bengaluru — high density since this is the test chef's city.
	{Code: "560001", CityID: "IN-KA-bengaluru", AreaName: "Bangalore GPO"},
	{Code: "560002", CityID: "IN-KA-bengaluru", AreaName: "Halasuru"},
	{Code: "560003", CityID: "IN-KA-bengaluru", AreaName: "Mathikere"},
	{Code: "560004", CityID: "IN-KA-bengaluru", AreaName: "Basavanagudi"},
	{Code: "560008", CityID: "IN-KA-bengaluru", AreaName: "Frazer Town"},
	{Code: "560011", CityID: "IN-KA-bengaluru", AreaName: "Banashankari"},
	{Code: "560020", CityID: "IN-KA-bengaluru", AreaName: "Rajajinagar"},
	{Code: "560025", CityID: "IN-KA-bengaluru", AreaName: "Bangalore Cantonment"},
	{Code: "560034", CityID: "IN-KA-bengaluru", AreaName: "Koramangala"},
	{Code: "560037", CityID: "IN-KA-bengaluru", AreaName: "Marathahalli"},
	{Code: "560038", CityID: "IN-KA-bengaluru", AreaName: "Indiranagar"},
	{Code: "560041", CityID: "IN-KA-bengaluru", AreaName: "Jayanagar"},
	{Code: "560066", CityID: "IN-KA-bengaluru", AreaName: "Whitefield"},
	{Code: "560068", CityID: "IN-KA-bengaluru", AreaName: "HSR Layout"},
	{Code: "560076", CityID: "IN-KA-bengaluru", AreaName: "BTM Layout"},
	{Code: "560078", CityID: "IN-KA-bengaluru", AreaName: "JP Nagar"},
	{Code: "560085", CityID: "IN-KA-bengaluru", AreaName: "Banashankari 3rd Stage"},
	{Code: "560095", CityID: "IN-KA-bengaluru", AreaName: "Domlur"},
	{Code: "560100", CityID: "IN-KA-bengaluru", AreaName: "Electronic City"},
	{Code: "560102", CityID: "IN-KA-bengaluru", AreaName: "Bellandur"},
	{Code: "560103", CityID: "IN-KA-bengaluru", AreaName: "Sarjapur Road"},

	// Mumbai
	{Code: "400001", CityID: "IN-MH-mumbai", AreaName: "Fort"},
	{Code: "400002", CityID: "IN-MH-mumbai", AreaName: "Kalbadevi"},
	{Code: "400005", CityID: "IN-MH-mumbai", AreaName: "Colaba"},
	{Code: "400020", CityID: "IN-MH-mumbai", AreaName: "Marine Lines"},
	{Code: "400028", CityID: "IN-MH-mumbai", AreaName: "Dadar"},
	{Code: "400049", CityID: "IN-MH-mumbai", AreaName: "Andheri West"},
	{Code: "400050", CityID: "IN-MH-mumbai", AreaName: "Bandra West"},
	{Code: "400051", CityID: "IN-MH-mumbai", AreaName: "Bandra East"},
	{Code: "400053", CityID: "IN-MH-mumbai", AreaName: "Andheri (Lokhandwala)"},
	{Code: "400055", CityID: "IN-MH-mumbai", AreaName: "Vile Parle East"},
	{Code: "400057", CityID: "IN-MH-mumbai", AreaName: "Vile Parle West"},
	{Code: "400063", CityID: "IN-MH-mumbai", AreaName: "Goregaon East"},
	{Code: "400064", CityID: "IN-MH-mumbai", AreaName: "Malad West"},
	{Code: "400070", CityID: "IN-MH-mumbai", AreaName: "Kurla West"},
	{Code: "400076", CityID: "IN-MH-mumbai", AreaName: "Powai"},

	// Pune
	{Code: "411001", CityID: "IN-MH-pune", AreaName: "Pune GPO"},
	{Code: "411004", CityID: "IN-MH-pune", AreaName: "Shivajinagar"},
	{Code: "411007", CityID: "IN-MH-pune", AreaName: "Aundh"},
	{Code: "411013", CityID: "IN-MH-pune", AreaName: "Wadgaon Sheri"},
	{Code: "411014", CityID: "IN-MH-pune", AreaName: "Viman Nagar"},
	{Code: "411038", CityID: "IN-MH-pune", AreaName: "Kothrud"},
	{Code: "411045", CityID: "IN-MH-pune", AreaName: "Baner"},
	{Code: "411057", CityID: "IN-MH-pune", AreaName: "Hinjewadi"},

	// Delhi
	{Code: "110001", CityID: "IN-DL-new-delhi", AreaName: "Connaught Place"},
	{Code: "110002", CityID: "IN-DL-new-delhi", AreaName: "Daryaganj"},
	{Code: "110003", CityID: "IN-DL-new-delhi", AreaName: "Lodhi Road"},
	{Code: "110005", CityID: "IN-DL-new-delhi", AreaName: "Karol Bagh"},
	{Code: "110017", CityID: "IN-DL-new-delhi", AreaName: "Hauz Khas"},
	{Code: "110019", CityID: "IN-DL-new-delhi", AreaName: "Kalkaji"},
	{Code: "110024", CityID: "IN-DL-new-delhi", AreaName: "Lajpat Nagar"},
	{Code: "110049", CityID: "IN-DL-new-delhi", AreaName: "Defence Colony"},
	{Code: "110057", CityID: "IN-DL-new-delhi", AreaName: "Vasant Vihar"},
	{Code: "110065", CityID: "IN-DL-new-delhi", AreaName: "Greater Kailash"},
	{Code: "110092", CityID: "IN-DL-new-delhi", AreaName: "Vasundhara Enclave"},

	// Chennai
	{Code: "600001", CityID: "IN-TN-chennai", AreaName: "Chennai GPO"},
	{Code: "600002", CityID: "IN-TN-chennai", AreaName: "Anna Salai"},
	{Code: "600004", CityID: "IN-TN-chennai", AreaName: "Mylapore"},
	{Code: "600020", CityID: "IN-TN-chennai", AreaName: "Adyar"},
	{Code: "600028", CityID: "IN-TN-chennai", AreaName: "Raja Annamalai Puram"},
	{Code: "600034", CityID: "IN-TN-chennai", AreaName: "Nungambakkam"},
	{Code: "600040", CityID: "IN-TN-chennai", AreaName: "Anna Nagar"},
	{Code: "600041", CityID: "IN-TN-chennai", AreaName: "Thiruvanmiyur"},
	{Code: "600042", CityID: "IN-TN-chennai", AreaName: "Velachery"},
	{Code: "600096", CityID: "IN-TN-chennai", AreaName: "Perungudi"},
	{Code: "600119", CityID: "IN-TN-chennai", AreaName: "Sholinganallur"},

	// Hyderabad
	{Code: "500001", CityID: "IN-TG-hyderabad", AreaName: "Hyderabad GPO"},
	{Code: "500003", CityID: "IN-TG-hyderabad", AreaName: "Secunderabad"},
	{Code: "500016", CityID: "IN-TG-hyderabad", AreaName: "Begumpet"},
	{Code: "500017", CityID: "IN-TG-hyderabad", AreaName: "Banjara Hills"},
	{Code: "500032", CityID: "IN-TG-hyderabad", AreaName: "Gachibowli"},
	{Code: "500033", CityID: "IN-TG-hyderabad", AreaName: "Jubilee Hills"},
	{Code: "500034", CityID: "IN-TG-hyderabad", AreaName: "Khairatabad"},
	{Code: "500081", CityID: "IN-TG-hyderabad", AreaName: "HITEC City"},
	{Code: "500084", CityID: "IN-TG-hyderabad", AreaName: "Madhapur"},

	// Kolkata
	{Code: "700001", CityID: "IN-WB-kolkata", AreaName: "Kolkata GPO"},
	{Code: "700016", CityID: "IN-WB-kolkata", AreaName: "Park Street"},
	{Code: "700017", CityID: "IN-WB-kolkata", AreaName: "Ballygunge"},
	{Code: "700019", CityID: "IN-WB-kolkata", AreaName: "Gariahat"},
	{Code: "700020", CityID: "IN-WB-kolkata", AreaName: "Alipore"},
	{Code: "700026", CityID: "IN-WB-kolkata", AreaName: "Bhowanipore"},
	{Code: "700064", CityID: "IN-WB-kolkata", AreaName: "Salt Lake"},
	{Code: "700091", CityID: "IN-WB-kolkata", AreaName: "Sector V Salt Lake"},
	{Code: "700156", CityID: "IN-WB-kolkata", AreaName: "Rajarhat"},

	// Ahmedabad
	{Code: "380001", CityID: "IN-GJ-ahmedabad", AreaName: "Ahmedabad GPO"},
	{Code: "380006", CityID: "IN-GJ-ahmedabad", AreaName: "Navrangpura"},
	{Code: "380009", CityID: "IN-GJ-ahmedabad", AreaName: "Bodakdev"},
	{Code: "380015", CityID: "IN-GJ-ahmedabad", AreaName: "Vastrapur"},
	{Code: "380054", CityID: "IN-GJ-ahmedabad", AreaName: "Satellite"},
	{Code: "380058", CityID: "IN-GJ-ahmedabad", AreaName: "Gota"},

	// Jaipur
	{Code: "302001", CityID: "IN-RJ-jaipur", AreaName: "Jaipur GPO"},
	{Code: "302004", CityID: "IN-RJ-jaipur", AreaName: "Civil Lines"},
	{Code: "302015", CityID: "IN-RJ-jaipur", AreaName: "Vaishali Nagar"},
	{Code: "302017", CityID: "IN-RJ-jaipur", AreaName: "Mansarovar"},
	{Code: "302020", CityID: "IN-RJ-jaipur", AreaName: "Malviya Nagar"},

	// Lucknow / Noida / Gurugram (NCR + UP)
	{Code: "201301", CityID: "IN-UP-noida", AreaName: "Noida Sector 18"},
	{Code: "201307", CityID: "IN-UP-noida", AreaName: "Greater Noida West"},
	{Code: "226001", CityID: "IN-UP-lucknow", AreaName: "Lucknow GPO"},
	{Code: "226010", CityID: "IN-UP-lucknow", AreaName: "Gomti Nagar"},
	{Code: "226024", CityID: "IN-UP-lucknow", AreaName: "Hazratganj"},
	{Code: "122002", CityID: "IN-HR-gurugram", AreaName: "Sector 17"},
	{Code: "122009", CityID: "IN-HR-gurugram", AreaName: "Sector 56"},
	{Code: "122018", CityID: "IN-HR-gurugram", AreaName: "DLF Phase 2"},

	// Indore / Bhopal
	{Code: "452001", CityID: "IN-MP-indore", AreaName: "Indore GPO"},
	{Code: "452010", CityID: "IN-MP-indore", AreaName: "Vijay Nagar"},
	{Code: "462001", CityID: "IN-MP-bhopal", AreaName: "Bhopal GPO"},
	{Code: "462016", CityID: "IN-MP-bhopal", AreaName: "Arera Colony"},

	// Kerala
	{Code: "682001", CityID: "IN-KL-kochi", AreaName: "Kochi GPO"},
	{Code: "682016", CityID: "IN-KL-kochi", AreaName: "Kakkanad"},
	{Code: "682020", CityID: "IN-KL-kochi", AreaName: "Edappally"},
	{Code: "695001", CityID: "IN-KL-thiruvananthapuram", AreaName: "Thiruvananthapuram GPO"},

	// Chandigarh / Punjab
	{Code: "160017", CityID: "IN-CH-chandigarh", AreaName: "Sector 17"},
	{Code: "160022", CityID: "IN-CH-chandigarh", AreaName: "Sector 22"},
	{Code: "141001", CityID: "IN-PB-ludhiana", AreaName: "Ludhiana GPO"},
	{Code: "143001", CityID: "IN-PB-amritsar", AreaName: "Amritsar GPO"},

	// Misc state capitals
	{Code: "751001", CityID: "IN-OR-bhubaneswar", AreaName: "Bhubaneswar GPO"},
	{Code: "800001", CityID: "IN-BR-patna", AreaName: "Patna GPO"},
	{Code: "781001", CityID: "IN-AS-guwahati", AreaName: "Guwahati GPO"},
	{Code: "248001", CityID: "IN-UK-dehradun", AreaName: "Dehradun GPO"},
	{Code: "492001", CityID: "IN-CT-raipur", AreaName: "Raipur GPO"},
	{Code: "834001", CityID: "IN-JH-ranchi", AreaName: "Ranchi GPO"},
	{Code: "403001", CityID: "IN-GA-panaji", AreaName: "Panaji"},
	{Code: "171001", CityID: "IN-HP-shimla", AreaName: "Shimla GPO"},

	// AP / coastal
	{Code: "530001", CityID: "IN-AP-visakhapatnam", AreaName: "Visakhapatnam GPO"},
	{Code: "520001", CityID: "IN-AP-vijayawada", AreaName: "Vijayawada GPO"},
}
