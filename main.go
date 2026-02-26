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
	"github.com/xuri/excelize/v2"
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

	url := "https://nloto.ru/lottery/mechtallion/rules"

	fmt.Printf("Переход на страницу: %s\n", url)

	// 5. Выполнение действий в браузере
	var items []struct {
		Prize string `json:"prize"`
		Move  string `json:"move"`
	}

	err = chromedp.Run(ctx,
		chromedp.Navigate(url),
		// Ждем, пока элемент с нужным классом появится в DOM
		chromedp.WaitVisible(`.LQnNN`, chromedp.ByQuery),
		// Извлекаем данные через JS
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('.bpONIu')).map(el => {
				const prize = el.querySelector('.bzquVz')?.innerText || "";
				const move = el.querySelector('.jMNgrd')?.innerText || "";
				return { prize, move };
			})
		`, &items),
	)

	if err != nil {
		log.Fatal("Ошибка при выполнении chromedp:", err)
	}

	// 6. Формируем XLSX
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Println("Ошибка при закрытии файла XLSX:", err)
		}
	}()

	sheetName := "Results"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		log.Fatal("Не удалось создать лист в XLSX:", err)
	}
	f.SetActiveSheet(index)
	// Удаляем стандартный Sheet1, если он есть
	_ = f.DeleteSheet("Sheet1")

	// Заголовок
	_ = f.SetCellValue(sheetName, "A1", "Номер хода")
	_ = f.SetCellValue(sheetName, "B1", "Размер выигрыша")

	row := 2
	for _, item := range items {
		if item.Move != "" && item.Prize != "" {
			_ = f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), item.Move)
			_ = f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), item.Prize)
			row++
		}
	}

	if err := f.SaveAs("results.xlsx"); err != nil {
		log.Fatal("Не удалось сохранить XLSX файл:", err)
	}

	fmt.Printf("Успешно извлечено %d записей и сохранено в results.xlsx\n", len(items))
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
