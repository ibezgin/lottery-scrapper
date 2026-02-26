package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/playwright-community/playwright-go"
)

func main() {
	// 1. Скачиваем/убеждаемся в наличии браузера Chromium
	chromePath, err := setupBrowser()
	if err != nil {
		log.Fatal("Ошибка при настройке браузера:", err)
	}

	// 2. Опции запуска Chrome
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),   // Используем скачанный путь
		chromedp.Flag("headless", true), // Без оконного режима (headless)
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// 3. Создаем контекст chromedp
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	// 4. Устанавливаем таймаут на выполнение всей задачи
	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	var result string
	url := "https://nloto.ru/lottery/mechtallion/rules"

	fmt.Printf("Переход на страницу: %s\n", url)

	// 5. Выполнение действий в браузере
	err = chromedp.Run(ctx,
		chromedp.Navigate(url),
		// Ждем, пока элемент с нужным классом появится в DOM
		chromedp.WaitVisible(`.sc-b3db3955-0`, chromedp.ByQuery),
		// Извлекаем текст элемента
		chromedp.Text(`.sc-b3db3955-0`, &result, chromedp.ByQuery),
	)

	if err != nil {
		log.Fatal("Ошибка при выполнении chromedp:", err)
	}

	fmt.Printf("Найдено содержимое (sc-b3db3955-0):\n%s\n", result)
	fmt.Println("Скрапинг завершен.")
}

func setupBrowser() (string, error) {
	fmt.Println("Проверка браузера...")

	// Настраиваем Playwright для скачивания в текущую директорию
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	browserDir := filepath.Join(cwd, "browser")
	if _, err := os.Stat(browserDir); os.IsNotExist(err) {
		if err := os.MkdirAll(browserDir, 0755); err != nil {
			return "", err
		}
	}

	// Переменная окружения PLAYWRIGHT_BROWSERS_PATH указывает Playwright куда скачивать
	os.Setenv("PLAYWRIGHT_BROWSERS_PATH", browserDir)

	// Устанавливаем драйвер, если его нет
	err = playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
	})
	if err != nil {
		return "", fmt.Errorf("ошибка при скачивании браузера: %w", err)
	}

	// Пытаемся найти путь к исполняемому файлу
	pw, err := playwright.Run()
	if err != nil {
		return "", err
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		return "", err
	}
	// В Playwright-go путь к исполняемому файлу можно получить через BrowserType
	executablePath := pw.Chromium.ExecutablePath()
	_ = browser.Close()

	fmt.Printf("Используется браузер: %s\n", executablePath)
	return executablePath, nil
}
