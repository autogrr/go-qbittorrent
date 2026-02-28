package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	qbt "github.com/autogrr/go-qbittorrent"
)

func main() {
	client, err := qbt.NewClient(qbt.Config{
		Host:     envOr("QBT_HOST", "http://localhost:8080"),
		Username: envOr("QBT_USER", "admin"),
		Password: envOr("QBT_PASS", "adminadmin"),
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	if err := client.Login(ctx); err != nil {
		log.Fatalf("login: %v", err)
	}

	ver, _ := client.GetAppVersion(ctx)
	api, _ := client.GetWebAPIVersion(ctx)
	fmt.Println("qBittorrent", ver, "API", api)

	info, _ := client.GetTransferInfo(ctx)
	fmt.Printf("DOWN %d KiB/s  UP %d KiB/s\n", qbt.Deref(info.DlInfoSpeed)/1024, qbt.Deref(info.UpInfoSpeed)/1024)

	torrents, _ := client.GetTorrents(ctx, qbt.TorrentFilterOptions{Filter: qbt.FilterAll, Limit: 5})
	for _, t := range torrents {
		fmt.Printf("  %-30s %.0f%%  %s\n", qbt.Deref(t.Name), qbt.Deref(t.Progress)*100, qbt.Deref(t.State))
	}

	// Background sync.
	sm := client.NewSyncManager(qbt.SyncOptions{
		Interval:        time.Second,
		DynamicInterval: true,
		OnUpdate: func(state *qbt.SyncState) {
			ss := state.GetServerState()
			fmt.Printf("[sync] n=%d free=%dMiB down=%dKiB/s\n",
				len(state.GetTorrents()), qbt.Deref(ss.FreeSpaceOnDisk)/(1<<20), qbt.Deref(ss.DlInfoSpeed)/1024)
		},
	})
	sm.Start(ctx)
	time.Sleep(5 * time.Second)
	sm.Stop()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
