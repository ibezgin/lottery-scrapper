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

type LotteryItem struct {
	Prize string `json:"prize"`
	Move  string `json:"move"`
}

func main() {
	// 1. Скачиваем/убеждаемся в наличии браузера Chromium
	chromePath, err := setupBrowser()
	if err != nil {
		log.Fatal("Ошибка при настройке браузера:", err)
	}

	// 2. Настраиваем контекст chromedp
	ctx, allocCancel, ctxCancel := getChromedpContext(chromePath)
	defer allocCancel()
	defer ctxCancel()

	url := "https://nloto.ru/lottery/mechtallion/rules"

	// 3. Выполнение действий в браузере
	items, err := scrapeLotteryData(ctx, url)
	if err != nil {
		log.Fatal("Ошибка при выполнении chromedp:", err)
	}

	// 4. Сохраняем в XLSX
	filename := "results.xlsx"
	if err := saveToExcel(items, filename); err != nil {
		log.Fatal("Ошибка при сохранении XLSX:", err)
	}

	fmt.Printf("Успешно извлечено %d записей и сохранено в %s\n", len(items), filename)
	fmt.Println("Скрапинг завершен.")
}

func getChromedpContext(chromePath string) (context.Context, context.CancelFunc, context.CancelFunc) {
	// Опции запуска Chrome
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(chromePath),   // Используем скачанный путь
		chromedp.Flag("headless", true), // Без оконного режима (headless)
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	// Создаем контекст chromedp
	ctx, ctxCancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)

	// Устанавливаем таймаут на выполнение всей задачи
	ctx, timeoutCancel := context.WithTimeout(ctx, 120*time.Second)

	combinedCtxCancel := func() {
		timeoutCancel()
		ctxCancel()
	}

	return ctx, allocCancel, combinedCtxCancel
}

func scrapeLotteryData(ctx context.Context, url string) ([]LotteryItem, error) {
	fmt.Printf("Переход на страницу: %s\n", url)

	var items []LotteryItem
	err := chromedp.Run(ctx,
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

	return items, err
}

func saveToExcel(items []LotteryItem, filename string) error {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Println("Ошибка при закрытии файла XLSX:", err)
		}
	}()

	sheetName := "Results"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("не удалось создать лист в XLSX: %w", err)
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

	if err := f.SaveAs(filename); err != nil {
		return fmt.Errorf("не удалось сохранить XLSX файл: %w", err)
	}

	return nil
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
