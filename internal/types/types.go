package types

// Game phases
type GamePhase string

const (
	PhaseTitle             GamePhase = "title"
	PhaseOssuary           GamePhase = "ossuary"
	PhaseGameSetup         GamePhase = "game_setup"
	PhaseCharacterCreation GamePhase = "character_creation"
	PhaseExploring         GamePhase = "exploring"
	PhaseCombat            GamePhase = "combat"
	PhaseLooting           GamePhase = "looting"
	PhaseDead              GamePhase = "dead"
	PhaseVictory           GamePhase = "victory"
)

type GameType string

const (
	GameTypeSprint     GameType = "sprint"
	GameTypeExpedition GameType = "expedition"
)

const MaxSlots = 10

// Content types

type Monster struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Str        int      `json:"str"`
	Dex        int      `json:"dex"`
	Wil        int      `json:"wil"`
	HP         int      `json:"hp"`
	Armor      int      `json:"armor"`
	Attack     Attack   `json:"attack"`
	Special    *string  `json:"special"`
	Tier       string   `json:"tier"`
	Weaknesses []string `json:"weaknesses"`
	Image      string   `json:"image,omitempty"`
}

type Attack struct {
	Name string `json:"name"`
	Die  string `json:"die"`
}

type UseEffect struct {
	Type  string `json:"type"`
	Die   string `json:"die,omitempty"`
	Level int    `json:"level,omitempty"`
}

type Item struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Slots       int        `json:"slots"`
	Damage      string     `json:"damage,omitempty"`
	ArmorValue  int        `json:"armorValue,omitempty"`
	Description string     `json:"description,omitempty"`
	Traits      []string   `json:"traits,omitempty"`
	UseEffect   *UseEffect `json:"useEffect,omitempty"`
}

type RoomFeature struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
}

type EncounterDef struct {
	Type   string  `json:"type"`
	Tier   string  `json:"tier"`
	Chance float64 `json:"chance"`
}

type ExitDef struct {
	Label     string `json:"label"`
	Direction string `json:"direction"`
}

type LootDef struct {
	Tier   string  `json:"tier"`
	Chance float64 `json:"chance"`
}

type RoomTemplate struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Features   []RoomFeature `json:"features"`
	Encounters []EncounterDef `json:"encounters"`
	Exits      []ExitDef     `json:"exits"`
	Loot       *LootDef      `json:"loot"`
}

type FragmentPool map[string]map[string][]string

type StartingGearTables struct {
	Weapons  []Item `json:"weapons"`
	Armor    []Item `json:"armor"`
	Gear     []Item `json:"gear"`
	Trinkets []Item `json:"trinkets"`
}

type ContentData struct {
	Monsters     []Monster          `json:"monsters"`
	Items        []Item             `json:"items"`
	Rooms        []RoomTemplate     `json:"rooms"`
	Fragments    FragmentPool       `json:"fragments"`
	StartingGear StartingGearTables `json:"startingGear"`
	Names        []string           `json:"names"`
}

type ContentPackMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Created     string `json:"created"`
	Version     int    `json:"version"`
}

type ContentPack struct {
	Meta             ContentPackMeta    `json:"meta"`
	Supports         []GameType         `json:"supports,omitempty"`
	SprintConfig     *SprintConfig      `json:"sprintConfig,omitempty"`
	ExpeditionConfig *ExpeditionConfig  `json:"expeditionConfig,omitempty"`
	GmPrompt         string             `json:"gmPrompt"`
	Monsters         []Monster          `json:"monsters"`
	Items            []Item             `json:"items"`
	Rooms            []RoomTemplate     `json:"rooms"`
	Fragments        FragmentPool       `json:"fragments"`
	StartingGear     StartingGearTables `json:"startingGear"`
	Names            []string           `json:"names"`
}

type SprintConfig struct {
	Rooms int `json:"rooms,omitempty"`
}

type ExpeditionConfig struct {
	RoomBudget           [2]int  `json:"roomBudget"`
	BranchProbability    float64 `json:"branchProbability"`
	LoopProbability      float64 `json:"loopProbability"`
	DeadEndRatio         float64 `json:"deadEndRatio"`
	LightSourceFrequency float64 `json:"lightSourceFrequency"`
	GridSize             int     `json:"gridSize"`
}

// Game state types

type Scar struct {
	Description string `json:"description"`
	Effect      string `json:"effect"`
}

type Character struct {
	Name      string  `json:"name"`
	Str       int     `json:"str"`
	Dex       int     `json:"dex"`
	Wil       int     `json:"wil"`
	HP        int     `json:"hp"`
	MaxHP     int     `json:"maxHp"`
	Armor     int     `json:"armor"`
	Inventory []*Item `json:"inventory"`
	Scars     []Scar  `json:"scars"`
}

type MonsterInstance struct {
	Base       Monster `json:"base"`
	CurrentHP  int     `json:"currentHp"`
	CurrentStr int     `json:"currentStr"`
}

type Encounter struct {
	Type        string           `json:"type"`
	Monster     *MonsterInstance `json:"monster,omitempty"`
	Damage      string           `json:"damage,omitempty"`
	SaveStat    string           `json:"saveStat,omitempty"`
	Description string           `json:"description,omitempty"`
}

type DungeonRoom struct {
	TemplateID  string     `json:"templateId"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Encounter   *Encounter `json:"encounter"`
	Loot        []Item     `json:"loot"`
	Exits       []ExitDef  `json:"exits"`
	Visited     bool       `json:"visited"`
	Cleared     bool       `json:"cleared"`
}

type CombatState struct {
	Monster            MonsterInstance `json:"monster"`
	Round              int            `json:"round"`
	Log                []string       `json:"log"`
	MonsterStunned     bool           `json:"monsterStunned"`
	PlayerHasAdvantage bool           `json:"playerHasAdvantage"`
}

type LogEntry struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type Pos struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Walls struct {
	N bool `json:"n"`
	E bool `json:"e"`
	S bool `json:"s"`
	W bool `json:"w"`
}

type ExpeditionRoom struct {
	DungeonRoom
	Pos   Pos   `json:"pos"`
	Walls Walls `json:"walls"`
}

type SprintDungeon struct {
	Rooms            []DungeonRoom `json:"rooms"`
	CurrentRoomIndex int           `json:"currentRoomIndex"`
}

type ExpeditionDungeon struct {
	Grid        [][]*ExpeditionRoom `json:"grid"`
	PlayerPos   Pos                 `json:"playerPos"`
	PreviousPos *Pos                `json:"previousPos"`
	Entry       Pos                 `json:"entry"`
	BossPos     Pos                 `json:"bossPos"`
	RoomCount   int                 `json:"roomCount"`
	Visited     []string            `json:"visited"`
	Peeked      []string            `json:"peeked"`
}

type GameState struct {
	Phase          GamePhase          `json:"phase"`
	GameType       GameType           `json:"gameType"`
	Character      *Character         `json:"character"`
	Sprint         *SprintDungeon     `json:"sprint"`
	Expedition     *ExpeditionDungeon `json:"expedition"`
	Combat         *CombatState       `json:"combat"`
	PendingLoot    []Item             `json:"pendingLoot"`
	Log            []LogEntry         `json:"log"`
	Seed           int                `json:"seed"`
	RngState       int                `json:"rngState"`
	RestsTaken     int                `json:"restsTaken"`
	MonstersKilled int                `json:"monstersKilled"`
	Light          int                `json:"light"`
}

// Derived state helpers

func (s *GameState) CurrentRoom() *DungeonRoom {
	if s.Sprint != nil {
		if s.Sprint.CurrentRoomIndex < len(s.Sprint.Rooms) {
			return &s.Sprint.Rooms[s.Sprint.CurrentRoomIndex]
		}
		return nil
	}
	if s.Expedition != nil {
		room := s.Expedition.Grid[s.Expedition.PlayerPos.Y][s.Expedition.PlayerPos.X]
		if room != nil {
			return &room.DungeonRoom
		}
	}
	return nil
}

func (s *GameState) Exits() []ExitDef {
	room := s.CurrentRoom()
	if room == nil {
		return nil
	}
	return room.Exits
}

func (s *GameState) TotalRooms() int {
	if s.Sprint != nil {
		return len(s.Sprint.Rooms)
	}
	if s.Expedition != nil {
		return s.Expedition.RoomCount
	}
	return 0
}

func (s *GameState) RoomsVisited() int {
	if s.Sprint != nil {
		return max(s.Sprint.CurrentRoomIndex+1, 1)
	}
	if s.Expedition != nil {
		return len(s.Expedition.Visited)
	}
	return 0
}

func (s *GameState) IsFinalRoom() bool {
	if s.Sprint != nil {
		return s.Sprint.CurrentRoomIndex == len(s.Sprint.Rooms)-1
	}
	if s.Expedition != nil {
		return s.Expedition.PlayerPos == s.Expedition.BossPos
	}
	return false
}

func (s *GameState) Depth() int {
	if s.Sprint != nil {
		return s.Sprint.CurrentRoomIndex + 1
	}
	if s.Expedition != nil {
		return len(s.Expedition.Visited)
	}
	return 0
}

// CreativeActionResult is the GM's response to a creative action.
type CreativeActionResult struct {
	Allowed          bool     `json:"allowed"`
	Reason           string   `json:"reason"`
	SaveStat         string   `json:"save_stat"`
	Modifier         int      `json:"modifier"`
	TraitMatches     []string `json:"trait_matches"`
	Effect           string   `json:"effect"`
	EffectDie        string   `json:"effect_die"`
	ConsumesItem     *string  `json:"consumes_item"`
	SuccessNarration string   `json:"success_narration"`
	FailureNarration string   `json:"failure_narration"`
}

// Action types
type Action struct {
	Type string

	Seed          int
	ExitIndex     int
	SlotIndex     int
	ItemIndex     int
	CreativeResult *CreativeActionResult
	SelectedType  GameType
}

// Action constructors
func NewGame(seed int) Action          { return Action{Type: "new_game", Seed: seed} }
func AcceptCharacter() Action          { return Action{Type: "accept_character"} }
func RerollCharacter() Action          { return Action{Type: "reroll_character"} }
func ChooseExit(index int) Action      { return Action{Type: "choose_exit", ExitIndex: index} }
func AttackAction() Action             { return Action{Type: "attack"} }
func FleeAction() Action               { return Action{Type: "flee"} }
func RunPast() Action                  { return Action{Type: "run_past"} }
func UseItem(slot int) Action          { return Action{Type: "use_item", SlotIndex: slot} }
func TakeItem(index int) Action        { return Action{Type: "take_item", ItemIndex: index} }
func DropItem(slot int) Action         { return Action{Type: "drop_item", SlotIndex: slot} }
func EquipItem(slot int) Action        { return Action{Type: "equip_item", SlotIndex: slot} }
func SkipLoot() Action                 { return Action{Type: "skip_loot"} }
func Rest() Action                     { return Action{Type: "rest"} }
func Restart() Action                  { return Action{Type: "restart"} }
func OpenOssuary() Action              { return Action{Type: "open_ossuary"} }
func ReturnToTitle() Action            { return Action{Type: "return_to_title"} }
func StartGameSetup() Action           { return Action{Type: "start_game_setup"} }
func SelectType(gt GameType) Action    { return Action{Type: "select_type", SelectedType: gt} }
func CreativeAction(r *CreativeActionResult) Action {
	return Action{Type: "creative_action_result", CreativeResult: r}
}

// Run stats types

type RunRecord struct {
	Name          string  `json:"name"`
	Depth         int     `json:"depth"`
	TotalRooms    int     `json:"totalRooms"`
	MonstersKilled int    `json:"monstersKilled"`
	Result        string  `json:"result"`
	CauseOfDeath  *string `json:"causeOfDeath"`
	PackName      string  `json:"packName"`
	Timestamp     int64   `json:"timestamp"`
}

type PlayerStats struct {
	Runs                []RunRecord `json:"runs"`
	TotalRuns           int         `json:"totalRuns"`
	Victories           int         `json:"victories"`
	BestDepth           int         `json:"bestDepth"`
	TotalMonstersKilled int         `json:"totalMonstersKilled"`
}

type OssuaryStats struct {
	TotalRuns           int                `json:"totalRuns"`
	TotalVictories      int                `json:"totalVictories"`
	TotalDeaths         int                `json:"totalDeaths"`
	AverageDepth        float64            `json:"averageDepth"`
	Deadliest           []DeadliestEntry   `json:"deadliest"`
	Packs               []PackStats        `json:"packs"`
	TotalMonstersKilled int                `json:"totalMonstersKilled"`
	AverageDuration     *float64           `json:"averageDuration"`
}

type DeadliestEntry struct {
	Name  string  `json:"name"`
	Kills int     `json:"kills"`
	Tier  *string `json:"tier"`
}

type PackStats struct {
	Name      string `json:"name"`
	Runs      int    `json:"runs"`
	Victories int    `json:"victories"`
	WinRate   int    `json:"winRate"`
}

// LightBand helpers

type LightBand string

const (
	LightBright LightBand = "bright"
	LightDim    LightBand = "dim"
	LightDark   LightBand = "dark"
	LightBlack  LightBand = "black"
)

func GetLightBand(light int) LightBand {
	switch {
	case light >= 7:
		return LightBright
	case light >= 3:
		return LightDim
	case light >= 1:
		return LightDark
	default:
		return LightBlack
	}
}
