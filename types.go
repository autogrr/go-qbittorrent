package qbittorrent

// TorrentState represents the state of a torrent.
type TorrentState string

const (
	StateError              TorrentState = "error"
	StateMissingFiles       TorrentState = "missingFiles"
	StateUploading          TorrentState = "uploading"
	StatePausedUP           TorrentState = "pausedUP"
	StateStoppedUP          TorrentState = "stoppedUP"
	StateQueuedUP           TorrentState = "queuedUP"
	StateStalledUP          TorrentState = "stalledUP"
	StateCheckingUP         TorrentState = "checkingUP"
	StateForcedUP           TorrentState = "forcedUP"
	StateAllocating         TorrentState = "allocating"
	StateDownloading        TorrentState = "downloading"
	StateMetaDL             TorrentState = "metaDL"
	StatePausedDL           TorrentState = "pausedDL"
	StateStoppedDL          TorrentState = "stoppedDL"
	StateQueuedDL           TorrentState = "queuedDL"
	StateStalledDL          TorrentState = "stalledDL"
	StateCheckingDL         TorrentState = "checkingDL"
	StateForcedDL           TorrentState = "forcedDL"
	StateCheckingResumeData TorrentState = "checkingResumeData"
	StateMoving             TorrentState = "moving"
	StateUnknown            TorrentState = "unknown"
)

func (s TorrentState) IsDownloading() bool {
	switch s {
	case StateDownloading, StateMetaDL, StateStalledDL, StateQueuedDL,
		StateForcedDL, StateCheckingDL, StateAllocating:
		return true
	}
	return false
}

func (s TorrentState) IsUploading() bool {
	switch s {
	case StateUploading, StateStalledUP, StateQueuedUP, StateForcedUP:
		return true
	}
	return false
}

func (s TorrentState) IsPaused() bool {
	return s == StatePausedDL || s == StatePausedUP ||
		s == StateStoppedDL || s == StateStoppedUP
}

func (s TorrentState) IsComplete() bool {
	return s == StateUploading || s == StateStalledUP ||
		s == StateForcedUP || s == StateQueuedUP || s == StateCheckingUP
}

// TorrentFilter is the filter to apply when listing torrents.
type TorrentFilter string

const (
	FilterAll              TorrentFilter = "all"
	FilterDownloading      TorrentFilter = "downloading"
	FilterSeeding          TorrentFilter = "seeding"
	FilterCompleted        TorrentFilter = "completed"
	FilterPaused           TorrentFilter = "paused"
	FilterStopped          TorrentFilter = "stopped"
	FilterActive           TorrentFilter = "active"
	FilterInactive         TorrentFilter = "inactive"
	FilterResumed          TorrentFilter = "resumed"
	FilterRunning          TorrentFilter = "running"
	FilterStalled          TorrentFilter = "stalled"
	FilterStalledUploading TorrentFilter = "stalled_uploading"
	FilterStalledDownload  TorrentFilter = "stalled_downloading"
	FilterErrored          TorrentFilter = "errored"
	FilterChecking         TorrentFilter = "checking"
	FilterMoving           TorrentFilter = "moving"
)

// ContentLayout controls the directory layout when adding a torrent.
type ContentLayout string

const (
	ContentLayoutOriginal      ContentLayout = "Original"
	ContentLayoutSubfolderNone ContentLayout = "NoSubfolder"
	ContentLayoutSubfolderOne  ContentLayout = "Subfolder"
)

// TrackerStatus represents the working status of a tracker.
type TrackerStatus int

const (
	TrackerDisabled     TrackerStatus = 0
	TrackerNotContacted TrackerStatus = 1
	TrackerWorking      TrackerStatus = 2
	TrackerUpdating     TrackerStatus = 3
	TrackerNotWorking   TrackerStatus = 4
)

// FilePriority controls how a file within a torrent is downloaded.
type FilePriority int

const (
	FilePrioritySkipped FilePriority = 0
	FilePriorityNormal  FilePriority = 1
	FilePriorityHigh    FilePriority = 6
	FilePriorityMaximal FilePriority = 7
)

// Torrent holds the list-level properties of a torrent (from torrents/info).
// All fields use pointer types so callers can distinguish a field that was
// absent from the JSON response (nil) from one that was present but zero.
type Torrent struct {
	AddedOn                  *int64        `json:"added_on"`
	AmountLeft               *int64        `json:"amount_left"`
	AutoTMM                  *bool         `json:"auto_tmm"`
	Availability             *float64      `json:"availability"`
	Category                 *string       `json:"category"`
	Completed                *int64        `json:"completed"`
	CompletionOn             *int64        `json:"completion_on"`
	ContentPath              *string       `json:"content_path"`
	DlLimit                  *int64        `json:"dl_limit"`
	DlSpeed                  *int64        `json:"dlspeed"`
	Downloaded               *int64        `json:"downloaded"`
	DownloadedSession        *int64        `json:"downloaded_session"`
	ETA                      *int64        `json:"eta"`
	FirstLastPiecePrio       *bool         `json:"f_l_piece_prio"`
	ForceStart               *bool         `json:"force_start"`
	Hash                     *string       `json:"hash"`
	InfoHashV1               *string       `json:"infohash_v1"`
	InfoHashV2               *string       `json:"infohash_v2"`
	LastActivity             *int64        `json:"last_activity"`
	MagnetURI                *string       `json:"magnet_uri"`
	MaxRatio                 *float64      `json:"max_ratio"`
	MaxSeedingTime           *int64        `json:"max_seeding_time"`
	Name                     *string       `json:"name"`
	NumComplete              *int          `json:"num_complete"`
	NumIncomplete            *int          `json:"num_incomplete"`
	NumLeechs                *int          `json:"num_leechs"`
	NumSeeds                 *int          `json:"num_seeds"`
	Priority                 *int          `json:"priority"`
	Progress                 *float64      `json:"progress"`
	Ratio                    *float64      `json:"ratio"`
	RatioLimit               *float64      `json:"ratio_limit"`
	SavePath                 *string       `json:"save_path"`
	DownloadPath             *string       `json:"download_path"`
	SeedingTime              *int64        `json:"seeding_time"`
	SeedingTimeLimit         *int64        `json:"seeding_time_limit"`
	SeenComplete             *int64        `json:"seen_complete"`
	SequentialDownload       *bool         `json:"seq_dl"`
	Size                     *int64        `json:"size"`
	State                    *TorrentState `json:"state"`
	SuperSeeding             *bool         `json:"super_seeding"`
	Tags                     *string       `json:"tags"`
	TimeActive               *int64        `json:"time_active"`
	TotalSize                *int64        `json:"total_size"`
	Tracker                  *string       `json:"tracker"`
	TrackersCount            *int          `json:"trackers_count"`
	UpLimit                  *int64        `json:"up_limit"`
	Uploaded                 *int64        `json:"uploaded"`
	UploadedSession          *int64        `json:"uploaded_session"`
	UpSpeed                  *int64        `json:"upspeed"`
	InactiveSeedingTimeLimit *int64        `json:"inactive_seeding_time_limit"`
	Reannounce               *int64        `json:"reannounce"`
	PopularityScore          *float64      `json:"popularity"`
	Private                  *bool         `json:"private"`
	// Files is populated when TorrentFilterOptions.IncludeFiles is true (qBittorrent >= 5.2).
	Files []TorrentFile `json:"files,omitempty"`
	// Trackers is populated when TorrentFilterOptions.IncludeTrackers is true (qBittorrent >= 5.1).
	Trackers []TorrentTracker `json:"trackers,omitempty"`
}

// TorrentProperties contains detailed properties for a single torrent.
type TorrentProperties struct {
	AdditionDate           *int64   `json:"addition_date"`
	Comment                *string  `json:"comment"`
	CompletionDate         *int64   `json:"completion_date"`
	CreatedBy              *string  `json:"created_by"`
	CreationDate           *int64   `json:"creation_date"`
	DlLimit                *int64   `json:"dl_limit"`
	DlSpeed                *int64   `json:"dl_speed"`
	DlSpeedAvg             *int64   `json:"dl_speed_avg"`
	DownloadPath           *string  `json:"download_path"`
	ETA                    *int64   `json:"eta"`
	Hash                   *string  `json:"hash"`
	InfoHashV1             *string  `json:"infohash_v1"`
	InfoHashV2             *string  `json:"infohash_v2"`
	IsPrivate              *bool    `json:"is_private"`
	LastSeen               *int64   `json:"last_seen"`
	Name                   *string  `json:"name"`
	NbConnections          *int     `json:"nb_connections"`
	NbConnectionsLimit     *int     `json:"nb_connections_limit"`
	Peers                  *int     `json:"peers"`
	PeersTotal             *int     `json:"peers_total"`
	PieceSize              *int64   `json:"piece_size"`
	PiecesHave             *int     `json:"pieces_have"`
	PiecesNum              *int     `json:"pieces_num"`
	Reannounce             *int64   `json:"reannounce"`
	SavePath               *string  `json:"save_path"`
	SeedingTime            *int64   `json:"seeding_time"`
	Seeds                  *int     `json:"seeds"`
	SeedsTotal             *int     `json:"seeds_total"`
	ShareRatio             *float64 `json:"share_ratio"`
	TimeElapsed            *int64   `json:"time_elapsed"`
	TotalDownloaded        *int64   `json:"total_downloaded"`
	TotalDownloadedSession *int64   `json:"total_downloaded_session"`
	TotalSize              *int64   `json:"total_size"`
	TotalUploaded          *int64   `json:"total_uploaded"`
	TotalUploadedSession   *int64   `json:"total_uploaded_session"`
	TotalWasted            *int64   `json:"total_wasted"`
	UpLimit                *int64   `json:"up_limit"`
	UpSpeed                *int64   `json:"up_speed"`
	UpSpeedAvg             *int64   `json:"up_speed_avg"`
}

// TorrentFile represents a file within a torrent.
type TorrentFile struct {
	Availability *float64      `json:"availability"`
	Index        *int          `json:"index"`
	IsSeed       *bool         `json:"is_seed"`
	Name         *string       `json:"name"`
	PieceRange   []int         `json:"piece_range"`
	Priority     *FilePriority `json:"priority"`
	Progress     *float64      `json:"progress"`
	Size         *int64        `json:"size"`
}

// TrackerEndpoint holds per-endpoint statistics for a tracker tier.
// Each tracker URL may be served by multiple endpoints (e.g. UDP/TCP).
type TrackerEndpoint struct {
	Name          *string        `json:"name"`
	Updating      *bool          `json:"updating"`
	Status        *TrackerStatus `json:"status"`
	Message       *string        `json:"msg"`
	BTVersion     *int           `json:"bt_version"`
	NumPeers      *int           `json:"num_peers"`
	NumSeeds      *int           `json:"num_seeds"`
	NumLeeches    *int           `json:"num_leeches"`
	NumDownloaded *int           `json:"num_downloaded"`
	NextAnnounce  *int64         `json:"next_announce"`
	MinAnnounce   *int64         `json:"min_announce"`
}

// TorrentTracker represents a single tracker for a torrent.
type TorrentTracker struct {
	URL           *string           `json:"url"`
	Tier          *int              `json:"tier"`
	Updating      *bool             `json:"updating"`
	Status        *TrackerStatus    `json:"status"`
	Message       *string           `json:"msg"`
	NumPeers      *int              `json:"num_peers"`
	NumSeeds      *int              `json:"num_seeds"`
	NumLeeches    *int              `json:"num_leeches"`
	NumDownloaded *int              `json:"num_downloaded"`
	// NextAnnounce is seconds since epoch of the next announce time.
	NextAnnounce  *int64            `json:"next_announce"`
	// MinAnnounce is seconds since epoch of the minimum announce time.
	MinAnnounce   *int64            `json:"min_announce"`
	// Endpoints holds per-endpoint details; only present on the /torrents/trackers
	// and /torrents/info (with includeTrackers=true) endpoints.
	Endpoints     []TrackerEndpoint `json:"endpoints,omitempty"`
}

// WebSeed represents a web seed URL.
type WebSeed struct {
	URL string `json:"url"`
}

// Category represents a torrent category.
type Category struct {
	Name     string `json:"name"`
	SavePath string `json:"savePath"`
}

// TransferInfo holds global transfer statistics.
type TransferInfo struct {
	ConnectionStatus     *string `json:"connection_status"`
	DHTNodes             *int64  `json:"dht_nodes"`
	DlInfoData           *int64  `json:"dl_info_data"`
	DlInfoSpeed          *int64  `json:"dl_info_speed"`
	DlRateLimit          *int64  `json:"dl_rate_limit"`
	UpInfoData           *int64  `json:"up_info_data"`
	UpInfoSpeed          *int64  `json:"up_info_speed"`
	UpRateLimit          *int64  `json:"up_rate_limit"`
	UseAltSpeedLimits    *bool   `json:"use_alt_speed_limits"`
	RefreshInterval      *int    `json:"refresh_interval"`
	WriteCache           *int64  `json:"write_cache_overload"`
	ReadCache            *int64  `json:"read_cache_overload"`
	FreeSpace            *int64  `json:"free_space_on_disk"`
	GlobalRatio          *string `json:"global_ratio"`
	QueuedIOJobs         *int64  `json:"queued_io_jobs"`
	AverageTimeQueue     *int64  `json:"average_time_queue"`
	TotalBuffersSize     *int64  `json:"total_buffers_size"`
	TotalPeerConnections *int64  `json:"total_peer_connections"`
}

// ServerState is the server state returned by sync/maindata.
// All fields use pointer types to distinguish absent fields (nil) from
// present-but-zero fields in partial sync updates.
type ServerState struct {
	AllTimeDownload      *int64  `json:"alltime_dl"`
	AllTimeUpload        *int64  `json:"alltime_ul"`
	AverageTimeQueue     *int    `json:"average_time_queue"`
	ConnectionStatus     *string `json:"connection_status"`
	DHTNodes             *int64  `json:"dht_nodes"`
	DlInfoData           *int64  `json:"dl_info_data"`
	DlInfoSpeed          *int64  `json:"dl_info_speed"`
	DlRateLimit          *int64  `json:"dl_rate_limit"`
	FreeSpaceOnDisk      *int64  `json:"free_space_on_disk"`
	GlobalRatio          *string `json:"global_ratio"`
	GlobalSeedingTime    *int64  `json:"global_seeding_time"`
	QueuedIOJobs         *int64  `json:"queued_io_jobs"`
	Queueing             *bool   `json:"queueing"`
	ReadCacheHits        *string `json:"read_cache_hits"`
	ReadCacheOverload    *string `json:"read_cache_overload"`
	RefreshInterval      *int    `json:"refresh_interval"`
	TotalBuffersSize     *int64  `json:"total_buffers_size"`
	TotalPeerConnections *int64  `json:"total_peer_connections"`
	TotalQueuedSize      *int64  `json:"total_queued_size"`
	TotalWastedSession   *int64  `json:"total_wasted_session"`
	UpInfoData           *int64  `json:"up_info_data"`
	UpInfoSpeed          *int64  `json:"up_info_speed"`
	UpRateLimit          *int64  `json:"up_rate_limit"`
	UseAltSpeedLimits    *bool   `json:"use_alt_speed_limits"`
	WriteCacheOverload   *string `json:"write_cache_overload"`
}

// MainData is the response from sync/maindata.
type MainData struct {
	Rid               int                 `json:"rid"`
	FullUpdate        bool                `json:"full_update"`
	Torrents          map[string]Torrent  `json:"torrents"`
	TorrentsRemoved   []string            `json:"torrents_removed"`
	Categories        map[string]Category `json:"categories"`
	CategoriesRemoved []string            `json:"categories_removed"`
	Tags              []string            `json:"tags"`
	TagsRemoved       []string            `json:"tags_removed"`
	Trackers          map[string][]string `json:"trackers"`
	TrackersRemoved   []string            `json:"trackers_removed"`
	ServerState       ServerState         `json:"server_state"`
}

// LogEntry is a single log line from log/main.
type LogEntry struct {
	ID        *int    `json:"id"`
	Message   *string `json:"message"`
	Timestamp *int64  `json:"timestamp"`
	Type      *int    `json:"type"`
}

// PeerLogEntry is a peer log entry from log/peers.
type PeerLogEntry struct {
	Blocked   *bool   `json:"blocked"`
	ID        *int    `json:"id"`
	IP        *string `json:"ip"`
	Reason    *string `json:"reason"`
	Timestamp *int64  `json:"timestamp"`
}

// BuildInfo holds qBittorrent build metadata.
type BuildInfo struct {
	Bitness    *int    `json:"bitness"`
	Boost      *string `json:"boost"`
	LibTorrent *string `json:"libtorrent"`
	OpenSSL    *string `json:"openssl"`
	Platform   *string `json:"platform"`
	Qt         *string `json:"qt"`
	Zlib       *string `json:"zlib"`
}

// TorrentPeer holds data about a single peer.
// All scalar fields use pointer types. The Progress field being non-nil
// indicates the field was present in the JSON payload (replaces the old
// progressSet sentinel); use HasProgress() for a readable nil check.
type TorrentPeer struct {
	Client       *string  `json:"client"`
	Connection   *string  `json:"connection"`
	Country      *string  `json:"country"`
	CountryCode  *string  `json:"country_code"`
	DlSpeed      *int64   `json:"dl_speed"`
	Downloaded   *int64   `json:"downloaded"`
	Files        *string  `json:"files"`
	Flags        *string  `json:"flags"`
	FlagsDesc    *string  `json:"flags_desc"`
	IP           *string  `json:"ip"`
	PeerIDClient *string  `json:"peer_id_client"`
	Port         *int     `json:"port"`
	Progress     *float64 `json:"progress"`
	Relevance    *float64 `json:"relevance"`
	UpSpeed      *int64   `json:"up_speed"`
	Uploaded     *int64   `json:"uploaded"`
}

// HasProgress reports whether the progress field was present in the JSON payload.
func (tp TorrentPeer) HasProgress() bool { return tp.Progress != nil }

// TorrentPeersResponse is returned by sync/torrentPeers.
type TorrentPeersResponse struct {
	FullUpdate   bool                   `json:"full_update"`
	Peers        map[string]TorrentPeer `json:"peers"`
	PeersRemoved []string               `json:"peers_removed"`
	Rid          int                    `json:"rid"`
	ShowFlags    bool                   `json:"show_flags"`
}

// SearchJob represents a running search job.
type SearchJob struct {
	ID int `json:"id"`
}

// SearchStatus holds the status of a search job.
type SearchStatus struct {
	ID     *int    `json:"id"`
	Status *string `json:"status"`
	Total  *int    `json:"total"`
}

// SearchResult is a single result from a search.
type SearchResult struct {
	DescrLink  *string `json:"descrLink"`
	FileName   *string `json:"fileName"`
	FileSize   *int64  `json:"fileSize"`
	FileURL    *string `json:"fileUrl"`
	NbLeechers *int    `json:"nbLeechers"`
	NbSeeders  *int    `json:"nbSeeders"`
	SiteURL    *string `json:"siteUrl"`
}

// SearchResults is the full result set for a search query.
type SearchResults struct {
	Results []SearchResult `json:"results"`
	Status  *string        `json:"status"`
	Total   *int           `json:"total"`
}

// SearchPlugin is a qBittorrent search plugin descriptor.
type SearchPlugin struct {
	Enabled             bool   `json:"enabled"`
	FullName            string `json:"fullName"`
	Name                string `json:"name"`
	Namespace           string `json:"namespace"`
	SupportedCategories []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"supportedCategories"`
	URL     string `json:"url"`
	Version string `json:"version"`
}

// AppPreferences holds all qBittorrent application preferences.
// Only the most common fields are typed here; the full object is large.
type AppPreferences struct {
	AltDlLimit                         int64   `json:"alt_dl_limit"`
	AltUpLimit                         int64   `json:"alt_up_limit"`
	AlternativeWebUIEnabled            bool    `json:"alternative_webui_enabled"`
	AlternativeWebUIPath               string  `json:"alternative_webui_path"`
	AnnounceIP                         string  `json:"announce_ip"`
	AnnounceToAllTiers                 bool    `json:"announce_to_all_tiers"`
	AnnounceToAllTrackers              bool    `json:"announce_to_all_trackers"`
	AnonymousMode                      bool    `json:"anonymous_mode"`
	AsyncIOThreads                     int     `json:"async_io_threads"`
	AutoDeleteMode                     int     `json:"auto_delete_mode"`
	AutoTMMEnabled                     bool    `json:"auto_tmm_enabled"`
	AutorunEnabled                     bool    `json:"autorun_enabled"`
	AutorunProgram                     string  `json:"autorun_program"`
	BannedIPs                          string  `json:"banned_IPs"`
	BittorrentProtocol                 int     `json:"bittorrent_protocol"`
	CheckingMemoryUse                  int     `json:"checking_memory_use"`
	CreateTorrentSeedingMode           bool    `json:"create_torrent_seeding_mode"`
	CurrentInterfaceAddress            string  `json:"current_interface_address"`
	CurrentNetworkInterface            string  `json:"current_network_interface"`
	Dht                                bool    `json:"dht"`
	DlLimit                            int64   `json:"dl_limit"`
	DontCountSlowTorrents              bool    `json:"dont_count_slow_torrents"`
	DyndnsDomain                       string  `json:"dyndns_domain"`
	DyndnsEnabled                      bool    `json:"dyndns_enabled"`
	DyndnsPassword                     string  `json:"dyndns_password"`
	DyndnsService                      int     `json:"dyndns_service"`
	DyndnsUsername                     string  `json:"dyndns_username"`
	EmbeddedTrackerPort                int     `json:"embedded_tracker_port"`
	EmbeddedTrackerPortForwarding      bool    `json:"embedded_tracker_port_forwarding"`
	EnableCoalesceReadWrite            bool    `json:"enable_coalesce_read_write"`
	EnableEmbeddedTracker              bool    `json:"enable_embedded_tracker"`
	EnableMultiConnectionsFromSameIP   bool    `json:"enable_multi_connections_from_same_ip"`
	EnablePieceExtentAffinity          bool    `json:"enable_piece_extent_affinity"`
	EnableSuperSeeding                 bool    `json:"enable_super_seeding"`
	EnableUploadedSlot                 bool    `json:"enable_upload_slots"`
	Encryption                         int     `json:"encryption"`
	ExportDir                          string  `json:"export_dir"`
	ExportDirFin                       string  `json:"export_dir_fin"`
	FilePoolSize                       int     `json:"file_pool_size"`
	HashingThreads                     int     `json:"hashing_threads"`
	IDNSupportEnabled                  bool    `json:"idn_support_enabled"`
	IncompleteFilesExt                 bool    `json:"incomplete_files_ext"`
	IPFilterEnabled                    bool    `json:"ip_filter_enabled"`
	IPFilterPath                       string  `json:"ip_filter_path"`
	IPFilterTrackers                   bool    `json:"ip_filter_trackers"`
	IPRTT                              int     `json:"limit_utp_rate"`
	ListenPort                         int     `json:"listen_port"`
	Locale                             string  `json:"locale"`
	Lsd                                bool    `json:"lsd"`
	MailNotificationAuthEnabled        bool    `json:"mail_notification_auth_enabled"`
	MailNotificationEmail              string  `json:"mail_notification_email"`
	MailNotificationEnabled            bool    `json:"mail_notification_enabled"`
	MailNotificationPassword           string  `json:"mail_notification_password"`
	MailNotificationSMTP               string  `json:"mail_notification_smtp"`
	MailNotificationSSLEnabled         bool    `json:"mail_notification_ssl_enabled"`
	MailNotificationUsername           string  `json:"mail_notification_username"`
	MaxActiveCheckingTorrents          int     `json:"max_active_checking_torrents"`
	MaxActiveDownloads                 int     `json:"max_active_downloads"`
	MaxActiveTorrents                  int     `json:"max_active_torrents"`
	MaxActiveUploads                   int     `json:"max_active_uploads"`
	MaxConnec                          int     `json:"max_connec"`
	MaxConnecPerTorrent                int     `json:"max_connec_per_torrent"`
	MaxRatio                           float64 `json:"max_ratio"`
	MaxRatioAct                        int     `json:"max_ratio_act"`
	MaxRatioEnabled                    bool    `json:"max_ratio_enabled"`
	MaxSeedingTime                     int64   `json:"max_seeding_time"`
	MaxSeedingTimeEnabled              bool    `json:"max_seeding_time_enabled"`
	MaxUploads                         int     `json:"max_uploads"`
	MaxUploadsPerTorrent               int     `json:"max_uploads_per_torrent"`
	Pex                                bool    `json:"pex"`
	Preallocate                        bool    `json:"preallocate_all"`
	ProxyIP                            string  `json:"proxy_ip"`
	ProxyPassword                      string  `json:"proxy_password"`
	ProxyPeerConnections               bool    `json:"proxy_peer_connections"`
	ProxyPort                          int     `json:"proxy_port"`
	ProxyType                          int     `json:"proxy_type"`
	ProxyUsername                      string  `json:"proxy_username"`
	QueueingEnabled                    bool    `json:"queueing_enabled"`
	RandomPort                         bool    `json:"random_port"`
	RSSAutoDownloadingEnabled          bool    `json:"rss_auto_downloading_enabled"`
	RSSDownloadRepackProperEpisodes    bool    `json:"rss_download_repack_proper_episodes"`
	RSSMaxArticlesPerFeed              int     `json:"rss_max_articles_per_feed"`
	RSSProcessingEnabled               bool    `json:"rss_processing_enabled"`
	RSSRefreshInterval                 int     `json:"rss_refresh_interval"`
	RSSSmartEpisodeFilters             string  `json:"rss_smart_episode_filters"`
	ResumeDataStorageType              int     `json:"resume_data_storage_type"`
	SavePath                           string  `json:"save_path"`
	SaveResumeDataInterval             int     `json:"save_resume_data_interval"`
	ScanDirs                           any     `json:"scan_dirs"`
	ScheduleFrom                       int     `json:"schedule_from"`
	ScheduleTo                         int     `json:"schedule_to"`
	Scheduler                          bool    `json:"scheduler_enabled"`
	SchedulerDays                      int     `json:"scheduler_days"`
	SendBufferLoWatermark              int     `json:"send_buffer_low_watermark"`
	SendBufferWatermark                int     `json:"send_buffer_watermark"`
	SendBufferWatermarkFactor          int     `json:"send_buffer_watermark_factor"`
	SlowTorrentDLRateThreshold         int     `json:"slow_torrent_dl_rate_threshold"`
	SlowTorrentInactiveTimer           int     `json:"slow_torrent_inactive_timer"`
	SlowTorrentULRateThreshold         int     `json:"slow_torrent_ul_rate_threshold"`
	SocketBacklogSize                  int     `json:"socket_backlog_size"`
	SubcategoriesEnabled               bool    `json:"subcategories_enabled"`
	TempPath                           string  `json:"temp_path"`
	TempPathEnabled                    bool    `json:"temp_path_enabled"`
	TorrentChangedTMMEnabled           bool    `json:"torrent_changed_tmm_enabled"`
	TorrentContentLayout               string  `json:"torrent_content_layout"`
	TorrentStopCondition               string  `json:"torrent_stop_condition"`
	UpLimit                            int64   `json:"up_limit"`
	UploadChokingAlgorithm             int     `json:"upload_choking_algorithm"`
	UploadSlotsBehavior                int     `json:"upload_slots_behavior"`
	Upnp                               bool    `json:"upnp"`
	UseHTTPS                           bool    `json:"use_https"`
	UtpTCPMixedMode                    int     `json:"utp_tcp_mixed_mode"`
	WebUIDomainList                    string  `json:"web_ui_domain_list"`
	WebUIHTTPSCertPath                 string  `json:"web_ui_https_cert_path"`
	WebUIHTTPSKeyPath                  string  `json:"web_ui_https_key_path"`
	WebUIPort                          int     `json:"web_ui_port"`
	WebUIUpnp                          bool    `json:"web_ui_upnp"`
	WebUIUsername                      string  `json:"web_ui_username"`
	WebUICSRFProtectionEnabled         bool    `json:"web_ui_csrf_protection_enabled"`
	WebUIClickjackingProtectionEnabled bool    `json:"web_ui_clickjacking_protection_enabled"`
	WebUIHostHeaderValidationEnabled   bool    `json:"web_ui_host_header_validation_enabled"`
	WebUISecureCookieEnabled           bool    `json:"web_ui_secure_cookie_enabled"`
	WebUIMaxAuthFailCount              int     `json:"web_ui_max_auth_fail_count"`
	WebUIBanDuration                   int     `json:"web_ui_ban_duration"`
	WebUISessionTimeout                int     `json:"web_ui_session_timeout"`
}

// Cookie represents a browser-style cookie for tracker auth.
type Cookie struct {
	Domain         string  `json:"domain"`
	ExpirationDate float64 `json:"expirationDate"`
	Name           string  `json:"name"`
	Path           string  `json:"path"`
	Value          string  `json:"value"`
}

// TorrentCreationParams holds parameters for creating a new .torrent file (qBit ≥5.0).
type TorrentCreationParams struct {
	Format            string   `json:"format,omitempty"` // "v1", "v2", "hybrid"
	IsPrivate         bool     `json:"isPrivate,omitempty"`
	OptimizeAlignment bool     `json:"optimizeAlignment,omitempty"`
	PaddingEnabled    bool     `json:"paddingEnabled,omitempty"`
	PieceSize         int64    `json:"pieceSize,omitempty"` // in KiB; 0 = auto
	SourcePath        string   `json:"sourcePath"`
	Trackers          []string `json:"trackers,omitempty"`
	URLSeeds          []string `json:"urlSeeds,omitempty"`
	Comment           string   `json:"comment,omitempty"`
}

// TorrentCreationStatus is the result of GetTorrentCreationStatus.
type TorrentCreationStatus struct {
	ElapsedTime *int64   `json:"elapsedTime"`
	ID          *int     `json:"id"`
	Progress    *float64 `json:"progress"`
	Status      *string  `json:"status"` // "Running" | "Finished" | "Failed"
}

// RSSAutoDownloadRule is a qBittorrent RSS auto-download rule.
type RSSAutoDownloadRule struct {
	Enabled                   bool                  `json:"enabled"`
	MustContain               string                `json:"mustContain"`
	MustNotContain            string                `json:"mustNotContain"`
	UseRegex                  bool                  `json:"useRegex"`
	EpisodeFilter             string                `json:"episodeFilter"`
	SmartFilter               bool                  `json:"smartFilter"`
	PreviouslyMatchedEpisodes []string              `json:"previouslyMatchedEpisodes"`
	AffectedFeeds             []string              `json:"affectedFeeds"`
	IgnoreDays                int                   `json:"ignoreDays"`
	LastMatch                 string                `json:"lastMatch"`
	AddPaused                 *bool                 `json:"addPaused,omitempty"`
	AssignedCategory          string                `json:"assignedCategory"`
	SavePath                  string                `json:"savePath"`
	TorrentParams             *RSSRuleTorrentParams `json:"torrentParams,omitempty"`
}

// RSSRuleTorrentParams controls how matched RSS downloads are added.
type RSSRuleTorrentParams struct {
	Category            string        `json:"category,omitempty"`
	ContentLayout       ContentLayout `json:"contentLayout,omitempty"`
	DownloadLimit       int64         `json:"downloadLimit,omitempty"`
	DownloadPath        string        `json:"downloadPath,omitempty"`
	InactiveSeedingTime int           `json:"inactiveSeedingTimeLimit,omitempty"`
	OperatingMode       string        `json:"operatingMode,omitempty"`
	RatioLimit          float64       `json:"ratioLimit,omitempty"`
	SavePath            string        `json:"savePath,omitempty"`
	SeedingTimeLimit    int           `json:"seedingTimeLimit,omitempty"`
	SkipChecking        bool          `json:"skipChecking,omitempty"`
	Stopped             bool          `json:"stopped,omitempty"`
	Tags                []string      `json:"tags,omitempty"`
	UploadLimit         int64         `json:"uploadLimit,omitempty"`
	UseAutoTMM          *bool         `json:"useAutoTMM,omitempty"`
}
