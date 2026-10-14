package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/openact/formulae"
	"github.com/openact/kit/cli"
	"github.com/openact/kit/fs"
	_ "github.com/openact/modelSAA/lib"
	"go.uber.org/zap"
)

func main() {
	// Start CPU profiling
	f, err := os.Create("cpu.prof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	// Flags
	commonFlags := cli.RegisterCommonFlags()
	flag.Parse()

	// Resolve paths
	paths, err := commonFlags.Resolve()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// GetNum executable info (for logging)
	exePath, exeDir, err := fs.GetExecutableInfo()
	if err != nil {
		fmt.Printf("Error: Failed to get executable info: %v\n", err)
		os.Exit(1)
	}

	log.Printf("=== ExcelOperator Started ===")
	log.Printf("Executable path: %s", exePath)
	log.Printf("Executable directory: %s", exeDir)
	log.Printf("Working directory: %s", paths.WorkDir)
	log.Printf("config directory: %s", paths.ConfigDir)
	log.Printf("input directory: %s", paths.InputDir)
	log.Printf("output directory: %s", paths.OutputDir)

	// Existing code
	start := time.Now()

	// Build run settings（内部处理所有路径加载）
	settings, err := formulae.BuildRunSettings(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build run settings: %v\n", err)
		os.Exit(1)
	}

	// Run projection for each setting and each product component
	for _, s := range settings {
		// 为每个 setting 创建日志目录
		logdir := filepath.Join(s.ResultPath, ".log")
		if err := os.MkdirAll(logdir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create logs directory: %v\n", err)
			os.Exit(1)
		}

		// ✅ 修改点 1：创建独立的 logger（不是全局 logger）
		logPath := filepath.Join(logdir, s.OutputPaths.LogFile)

		settingLogger, err := formulae.NewSettingLogger("debug", true, logPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
			os.Exit(1)
		}
		defer settingLogger.Close() // ✅ 确保关闭文件句柄

		// ✅ 修改点 2：创建 context 并注入 logger
		ctx := &formulae.ProjContext{
			Setting: s,
		}
		ctx.SetLogger(settingLogger.Logger())

		// ✅ 修改点 3：使用 WithContext 记录日志
		formulae.WithContext(ctx).Info("=== START SETTING ===",
			zap.String("setting", s.Name))

		// ===== 以下保持原有逻辑不变 =====
		assemblySpec := filepath.Join(s.ConfPaths.StructureDir, s.Structure+".csv")
		structure, err := formulae.NewStructure(
			assemblySpec,
			formulae.Registry,
			s.ConfPaths.ProductDir,
			s.ConfPaths.AccumulationDir,
		)
		if err != nil {
			// ✅ 修改点 4：使用 WithContext 记录错误
			formulae.WithContext(ctx).Error("Failed to create structure",
				zap.String("setting", s.Name),
				zap.Error(err))
			os.Exit(1)
		}

		if err := formulae.RunProjection(ctx, s, structure); err != nil {
			formulae.WithContext(ctx).Error("Projection or output failed",
				zap.String("setting", s.Name), zap.Error(err))
			os.Exit(1)
		}

		// ✅ 修改点 5：使用 WithContext 记录完成信息
		formulae.WithContext(ctx).Info("All results processing completed for setting",
			zap.String("setting", s.Name))
	}

	timeLapse := time.Since(start)
	fmt.Printf("Projection completed in %v\n", timeLapse)
}
