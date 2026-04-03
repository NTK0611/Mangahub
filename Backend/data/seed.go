package main

import (
	"database/sql"
	"encoding/json"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type Manga struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	Genres        []string `json:"genres"`
	Status        string   `json:"status"`
	TotalChapters int      `json:"total_chapters"`
	Description   string   `json:"description"`
}

func main() {
	db, err := sql.Open("sqlite3", "./data/mangahub.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mangas := []Manga{
		// Shounen
		{ID: "one-piece", Title: "One Piece", Author: "Oda Eiichiro", Genres: []string{"Action", "Adventure", "Shounen"}, Status: "ongoing", TotalChapters: 1100, Description: "A young pirate's adventure to find the legendary treasure One Piece."},
		{ID: "naruto", Title: "Naruto", Author: "Kishimoto Masashi", Genres: []string{"Action", "Adventure", "Shounen"}, Status: "completed", TotalChapters: 700, Description: "A young ninja's journey to become the greatest Hokage."},
		{ID: "demon-slayer", Title: "Demon Slayer", Author: "Gotouge Koyoharu", Genres: []string{"Action", "Shounen"}, Status: "completed", TotalChapters: 205, Description: "A boy becomes a demon slayer to cure his demonified sister."},
		{ID: "my-hero-academia", Title: "My Hero Academia", Author: "Horikoshi Kouhei", Genres: []string{"Action", "Shounen"}, Status: "completed", TotalChapters: 430, Description: "A boy born without powers in a superhero-filled world."},
		{ID: "bleach", Title: "Bleach", Author: "Kubo Tite", Genres: []string{"Action", "Adventure", "Shounen"}, Status: "completed", TotalChapters: 686, Description: "A teenager gains the powers of a Soul Reaper."},
		{ID: "dragon-ball", Title: "Dragon Ball", Author: "Toriyama Akira", Genres: []string{"Action", "Adventure", "Shounen"}, Status: "completed", TotalChapters: 519, Description: "A boy with a monkey tail searches for the seven dragon balls."},
		{ID: "death-note", Title: "Death Note", Author: "Ohba Tsugumi", Genres: []string{"Mystery", "Thriller", "Shounen"}, Status: "completed", TotalChapters: 108, Description: "A student finds a supernatural notebook that kills anyone whose name is written in it."},
		{ID: "hunter-x-hunter", Title: "Hunter x Hunter", Author: "Togashi Yoshihiro", Genres: []string{"Action", "Adventure", "Shounen"}, Status: "ongoing", TotalChapters: 400, Description: "A boy searches for his father while becoming a legendary Hunter."},
		{ID: "fairy-tail", Title: "Fairy Tail", Author: "Mashima Hiro", Genres: []string{"Action", "Adventure", "Shounen"}, Status: "completed", TotalChapters: 545, Description: "A celestial wizard joins the most famous guild in the land."},
		{ID: "black-clover", Title: "Black Clover", Author: "Tabata Yuuki", Genres: []string{"Action", "Fantasy", "Shounen"}, Status: "ongoing", TotalChapters: 370, Description: "A boy born without magic aims to become the Wizard King."},
		{ID: "sword-art-online", Title: "Sword Art Online", Author: "Kawahara Reki", Genres: []string{"Action", "Adventure", "Shounen"}, Status: "ongoing", TotalChapters: 270, Description: "Players are trapped in a virtual reality MMORPG."},
		{ID: "seven-deadly-sins", Title: "Seven Deadly Sins", Author: "Suzuki Nakaba", Genres: []string{"Action", "Adventure", "Shounen"}, Status: "completed", TotalChapters: 346, Description: "A princess searches for the legendary knights to save her kingdom."},

		// Seinen
		{ID: "attack-on-titan", Title: "Attack on Titan", Author: "Isayama Hajime", Genres: []string{"Action", "Drama", "Seinen"}, Status: "completed", TotalChapters: 139, Description: "Humanity's last stand against giant man-eating titans."},
		{ID: "fullmetal-alchemist", Title: "Fullmetal Alchemist", Author: "Arakawa Hiromu", Genres: []string{"Action", "Adventure", "Seinen"}, Status: "completed", TotalChapters: 108, Description: "Two brothers use alchemy to search for the philosopher's stone."},
		{ID: "tokyo-ghoul", Title: "Tokyo Ghoul", Author: "Ishida Sui", Genres: []string{"Action", "Horror", "Seinen"}, Status: "completed", TotalChapters: 144, Description: "A student becomes half-ghoul after a deadly encounter in Tokyo."},
		{ID: "berserk", Title: "Berserk", Author: "Miura Kentaro", Genres: []string{"Action", "Dark Fantasy", "Seinen"}, Status: "ongoing", TotalChapters: 374, Description: "A lone mercenary struggles against demonic forces in a dark medieval world."},
		{ID: "vinland-saga", Title: "Vinland Saga", Author: "Yukimura Makoto", Genres: []string{"Action", "Adventure", "Seinen"}, Status: "ongoing", TotalChapters: 210, Description: "A young Viking warrior seeks revenge for his father's death."},
		{ID: "vagabond", Title: "Vagabond", Author: "Inoue Takehiko", Genres: []string{"Action", "Drama", "Seinen"}, Status: "ongoing", TotalChapters: 327, Description: "The story of legendary swordsman Miyamoto Musashi's journey."},
		{ID: "oyasumi-punpun", Title: "Goodnight Punpun", Author: "Asano Inio", Genres: []string{"Drama", "Psychological", "Seinen"}, Status: "completed", TotalChapters: 147, Description: "A coming-of-age story following a boy drawn as a small bird."},
		{ID: "gantz", Title: "Gantz", Author: "Oku Hiroya", Genres: []string{"Action", "Sci-Fi", "Seinen"}, Status: "completed", TotalChapters: 383, Description: "Dead people are brought back to hunt aliens in deadly missions."},

		// Shoujo
		{ID: "fruits-basket", Title: "Fruits Basket", Author: "Takaya Natsuki", Genres: []string{"Romance", "Drama", "Shoujo"}, Status: "completed", TotalChapters: 136, Description: "A girl discovers her classmates are possessed by the Chinese zodiac."},
		{ID: "sailor-moon", Title: "Sailor Moon", Author: "Takeuchi Naoko", Genres: []string{"Romance", "Fantasy", "Shoujo"}, Status: "completed", TotalChapters: 60, Description: "A clumsy schoolgirl transforms into a powerful guardian warrior."},
		{ID: "cardcaptor-sakura", Title: "Cardcaptor Sakura", Author: "CLAMP", Genres: []string{"Fantasy", "Romance", "Shoujo"}, Status: "completed", TotalChapters: 50, Description: "A young girl must recapture magical cards she accidentally released."},
		{ID: "skip-beat", Title: "Skip Beat!", Author: "Nakamura Yoshiki", Genres: []string{"Romance", "Drama", "Shoujo"}, Status: "ongoing", TotalChapters: 300, Description: "A girl enters showbiz to get revenge on her childhood friend."},
		{ID: "nana", Title: "Nana", Author: "Yazawa Ai", Genres: []string{"Romance", "Drama", "Shoujo"}, Status: "ongoing", TotalChapters: 84, Description: "Two girls named Nana meet on a train and become unlikely roommates."},

		// Josei
		{ID: "paradise-kiss", Title: "Paradise Kiss", Author: "Yazawa Ai", Genres: []string{"Romance", "Drama", "Josei"}, Status: "completed", TotalChapters: 40, Description: "A studious girl is recruited by a group of fashion design students."},
		{ID: "honey-and-clover", Title: "Honey and Clover", Author: "Umino Chica", Genres: []string{"Romance", "Drama", "Josei"}, Status: "completed", TotalChapters: 75, Description: "Art college students navigate love, friendship and career choices."},
		{ID: "josei-wotakoi", Title: "Wotakoi", Author: "Fujita", Genres: []string{"Romance", "Comedy", "Josei"}, Status: "completed", TotalChapters: 60, Description: "Two otaku adults start dating after reuniting at their workplace."},

		// Isekai / Fantasy
		{ID: "re-zero", Title: "Re:Zero", Author: "Nagatsuki Tappei", Genres: []string{"Fantasy", "Isekai", "Drama"}, Status: "ongoing", TotalChapters: 90, Description: "A boy transported to a fantasy world can only save others by dying."},
		{ID: "overlord", Title: "Overlord", Author: "Maruyama Kugane", Genres: []string{"Fantasy", "Isekai", "Action"}, Status: "ongoing", TotalChapters: 80, Description: "A player is trapped in an RPG as his powerful undead character."},
		{ID: "mushoku-tensei", Title: "Mushoku Tensei", Author: "Rifujin na Magonote", Genres: []string{"Fantasy", "Isekai", "Adventure"}, Status: "ongoing", TotalChapters: 90, Description: "A man reincarnates into a fantasy world and vows to live without regrets."},

		// Horror / Mystery
		{ID: "uzumaki", Title: "Uzumaki", Author: "Ito Junji", Genres: []string{"Horror", "Mystery", "Seinen"}, Status: "completed", TotalChapters: 20, Description: "A town becomes obsessed with spiral shapes in terrifying ways."},
		{ID: "monster", Title: "Monster", Author: "Urasawa Naoki", Genres: []string{"Mystery", "Thriller", "Seinen"}, Status: "completed", TotalChapters: 162, Description: "A doctor hunts down a patient he saved who became a serial killer."},
		{ID: "20th-century-boys", Title: "20th Century Boys", Author: "Urasawa Naoki", Genres: []string{"Mystery", "Sci-Fi", "Seinen"}, Status: "completed", TotalChapters: 249, Description: "A group of childhood friends try to stop a cult based on their old plans."},
		{ID: "parasyte", Title: "Parasyte", Author: "Iwaaki Hitoshi", Genres: []string{"Action", "Horror", "Seinen"}, Status: "completed", TotalChapters: 64, Description: "A boy's hand is taken over by an alien parasite that protects him."},
	}

	for _, m := range mangas {
		genresJSON, _ := json.Marshal(m.Genres)
		_, err := db.Exec(
			`INSERT OR IGNORE INTO manga (id, title, author, genres, status, total_chapters, description)
             VALUES (?, ?, ?, ?, ?, ?, ?)`,
			m.ID, m.Title, m.Author, string(genresJSON), m.Status, m.TotalChapters, m.Description,
		)
		if err != nil {
			log.Printf("Failed to insert %s: %v", m.Title, err)
		} else {
			log.Printf("✅ Inserted: %s", m.Title)
		}
	}
	log.Println("🎉 Seeding complete! 35 manga added.")
}
