// SPDX-FileCopyrightText: 2024 OOMOL, Inc. <https://www.oomol.com>
// SPDX-License-Identifier: MPL-2.0

package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/oomol-lab/ovm-win/pkg/logger"
	"github.com/oomol-lab/ovm-win/pkg/util"
)

func printPercent(ctx context.Context, log *logger.Context, file *os.File, total int64) error {
	if total == 0 {
		return nil
	}

	for {
		if ctx.Err() != nil {
			break
		}

		fi, err := file.Stat()
		if err != nil {
			if errors.Is(file.Close(), os.ErrClosed) {
				break
			}
			return fmt.Errorf("failed to get file info: %w", err)
		}

		size := fi.Size()
		if size == 0 {
			size = 1
		}

		log.Infof("Downloading %.2f%%", float64(size)/float64(total)*100)
		time.Sleep(time.Second)
	}

	return nil
}

func Download(ctx context.Context, log *logger.Context, url string, output string, sha256 string) error {
	if h, ok := util.Sha256File(output); ok && h == sha256 {
		log.Infof("File already downloaded, skip download")
		return nil
	} else if ok {
		log.Infof("Expected sha256: %s, but got %s", sha256, h)
	}

	tmpOutput := fmt.Sprintf("%s.tmp", output)
	if h, ok := util.Sha256File(tmpOutput); ok && h == sha256 {
		log.Infof("Temp file already downloaded, only rename")
		if err := os.Rename(tmpOutput, output); err != nil {
			return fmt.Errorf("failed to rename file: %w", err)
		}
		return nil
	}

	out, err := os.Create(tmpOutput)
	if err != nil {
		return fmt.Errorf("failed to create file in download: %w", err)
	}
	defer out.Close()

	headResp, err := http.Head(url)
	if err != nil {
		return fmt.Errorf("failed to send head request: %w", err)
	}
	contentLength := headResp.ContentLength

	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	context.AfterFunc(ctx, func() {
		log.Warn("Download canceled, because prent context is done")
		cancel()
	})

	go func() {
		if err := printPercent(ctx2, log, out, contentLength); err != nil {
			log.Warnf("Failed to calculate download percent: %v", err)
		}
	}()

	req, err := http.NewRequestWithContext(ctx2, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create get request: %w", err)
	}

	// TODO: need support multi-thread and resume download @BlackHole1
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send get request: %w", err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	_ = out.Close()
	_ = resp.Body.Close()

	cancel()

	if err := os.Rename(tmpOutput, output); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
