package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	pkgerrors "github.com/pkg/errors"

	"github.com/yachiyo/acgwarehouse/internal/conf"
	"github.com/yachiyo/acgwarehouse/internal/infra/db"
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/repository"
)

type tagOperation struct {
	addName        string
	addProvided    bool
	deleteName     string
	deleteProvided bool
	oldName        string
	updateProvided bool
	newName        string
}

type commandStreams struct {
	stdout io.Writer
	stderr io.Writer
}

type tagRunner struct {
	repo   *repository.TagRepository
	stdout io.Writer
}

func main() {
	streams := commandStreams{stdout: os.Stdout, stderr: os.Stderr}
	if err := run(context.Background(), os.Args[1:], streams); err != nil {
		if _, writeErr := fmt.Fprintf(os.Stderr, "tagctl: %v\n", err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

// run 解析命令行参数并执行标签管理操作。
func run(ctx context.Context, args []string, streams commandStreams) (err error) {
	operation, err := parseOperation(args, streams.stderr)
	if err != nil {
		return err
	}
	sqliteDB, err := db.NewSQLite(conf.LoadDatabase())
	if err != nil {
		return pkgerrors.WithMessage(err, "init sqlite")
	}
	defer func() {
		if closeErr := sqliteDB.Close(); closeErr != nil && err == nil {
			err = pkgerrors.WithMessage(closeErr, "close sqlite")
		}
	}()

	imageRepo := repository.NewImageRepository(sqliteDB.Read, sqliteDB.Write)
	tagRepo := repository.NewTagRepository(sqliteDB.Read, sqliteDB.Write, imageRepo)
	runner := tagRunner{repo: tagRepo, stdout: streams.stdout}
	return runner.execute(ctx, operation)
}

// parseOperation 将命令行参数解析为单一标签操作。
func parseOperation(args []string, stderr io.Writer) (tagOperation, error) {
	flags := flag.NewFlagSet("tagctl", flag.ContinueOnError)
	flags.SetOutput(stderr)
	addName := flags.String("a", "", "add tag by name")
	deleteName := flags.String("d", "", "delete tag by name")
	oldName := flags.String("u", "", "update tag by current name")
	newName := flags.String("name", "", "new tag name for update")
	if err := flags.Parse(args); err != nil {
		return tagOperation{}, pkgerrors.WithMessage(err, "parse flags")
	}
	operation := tagOperation{
		addName:        strings.TrimSpace(*addName),
		addProvided:    flagProvided(flags, "a"),
		deleteName:     strings.TrimSpace(*deleteName),
		deleteProvided: flagProvided(flags, "d"),
		oldName:        strings.TrimSpace(*oldName),
		updateProvided: flagProvided(flags, "u"),
		newName:        strings.TrimSpace(*newName),
	}
	if err := validateOperation(operation); err != nil {
		return tagOperation{}, err
	}
	return operation, nil
}

// flagProvided 判断调用方是否显式传入指定 flag。
func flagProvided(flags *flag.FlagSet, name string) bool {
	provided := false
	flags.Visit(func(flag *flag.Flag) {
		if flag.Name == name {
			provided = true
		}
	})
	return provided
}

// validateOperation 确保每次命令只包含一个完整操作。
func validateOperation(operation tagOperation) error {
	operationCount := 0
	if operation.addProvided {
		operationCount++
	}
	if operation.deleteProvided {
		operationCount++
	}
	if operation.updateProvided {
		operationCount++
	}
	if operationCount != 1 {
		return pkgerrors.New("provide exactly one of -a, -d, or -u")
	}
	if operation.updateProvided && operation.newName == "" {
		return pkgerrors.New("-name is required when using -u")
	}
	if !operation.updateProvided && operation.newName != "" {
		return pkgerrors.New("-name can only be used with -u")
	}
	return nil
}

// execute 调用标签仓储执行已解析的标签操作。
func (r tagRunner) execute(ctx context.Context, operation tagOperation) error {
	switch {
	case operation.addName != "":
		return r.add(ctx, operation.addName)
	case operation.deleteName != "":
		return r.delete(ctx, operation.deleteName)
	case operation.oldName != "":
		return r.update(ctx, operation)
	default:
		return pkgerrors.New("missing tag operation")
	}
}

// add 创建标签，名称已存在时复用已有标签。
func (r tagRunner) add(ctx context.Context, name string) error {
	tag, err := r.repo.Create(ctx, do.Tag{Name: name})
	if err != nil {
		return pkgerrors.WithMessage(err, "create tag")
	}
	_, err = fmt.Fprintf(r.stdout, "created tag %q (id=%d)\n", tag.Name, tag.ID)
	return pkgerrors.WithMessage(err, "write success message")
}

// delete 按名称查找标签并删除标签及其图片关联。
func (r tagRunner) delete(ctx context.Context, name string) error {
	tag, err := r.repo.FindByName(ctx, name)
	if err != nil {
		return pkgerrors.WithMessage(err, "find tag")
	}
	if err := r.repo.Delete(ctx, tag.ID); err != nil {
		return pkgerrors.WithMessage(err, "delete tag")
	}
	_, err = fmt.Fprintf(r.stdout, "deleted tag %q (id=%d)\n", tag.Name, tag.ID)
	return pkgerrors.WithMessage(err, "write success message")
}

// update 按旧名称查找标签并更新为新名称。
func (r tagRunner) update(ctx context.Context, operation tagOperation) error {
	tag, err := r.repo.FindByName(ctx, operation.oldName)
	if err != nil {
		return pkgerrors.WithMessage(err, "find tag")
	}
	updated, err := r.repo.Update(ctx, do.Tag{ID: tag.ID, Name: operation.newName})
	if err != nil {
		return pkgerrors.WithMessage(err, "update tag")
	}
	_, err = fmt.Fprintf(r.stdout, "updated tag %q to %q (id=%d)\n", tag.Name, updated.Name, updated.ID)
	return pkgerrors.WithMessage(err, "write success message")
}
