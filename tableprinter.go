package tableprinter

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/kataras/tablewriter"
)

// Alignment is the alignment type (int).
//
// See `Printer#DefaultColumnAlignment` and `Printer#DefaultColumnAlignment` too.
type Alignment int

const (
	// AlignDefault is the default alignment (0).
	AlignDefault Alignment = iota
	// AlignCenter is the center aligment (1).
	AlignCenter
	// AlignRight is the right aligment (2).
	AlignRight
	// AlignLeft is the left aligment (3).
	AlignLeft
)

// Printer contains some information about the final table presentation.
// Look its `Print` function for more.
type Printer struct {
	// out can not change during its work because the `acquire/release table` must work with only one output target,
	// a new printer should be declared for a different output.
	out io.Writer

	AutoFormatHeaders bool
	AutoWrapText      bool

	BorderTop, BorderLeft, BorderRight, BorderBottom bool

	HeaderLine      bool
	HeaderAlignment Alignment

	RowLine         bool
	ColumnSeparator string
	NewLine         string
	CenterSeparator string

	DefaultAlignment Alignment // see `NumbersAlignment` too.
	NumbersAlignment Alignment

	RowLengthTitle func(int) bool
	AllowRowsOnly  bool // if true then `Print/Render` will print the headers even if parsed rows where no found. Useful for putting rows to a table manually.

	table *tablewriter.Table
}

// Default is the default Table Printer.
var Default = Printer{
	out:               os.Stdout,
	AutoFormatHeaders: true,
	AutoWrapText:      false,

	BorderTop:    false,
	BorderLeft:   false,
	BorderRight:  false,
	BorderBottom: false,

	HeaderLine:      true,
	HeaderAlignment: AlignLeft,

	RowLine:         false, /* it could be true as well */
	ColumnSeparator: " ",
	NewLine:         "\n",
	CenterSeparator: " ", /* it could be empty as well */

	DefaultAlignment: AlignLeft,
	NumbersAlignment: AlignRight,

	RowLengthTitle: func(rowsLength int) bool {
		// if more than 3 then show the length of rows.
		return rowsLength > 3
	},

	AllowRowsOnly: true,
}

// New creates and initializes a Printer with the default values based on the "w" target writer.
//
// See its `Print`, `PrintHeadList` too.
func New(w io.Writer) *Printer {
	return &Printer{
		out: w,

		AutoFormatHeaders: Default.AutoFormatHeaders,
		AutoWrapText:      Default.AutoWrapText,

		BorderTop:    Default.BorderTop,
		BorderLeft:   Default.BorderLeft,
		BorderRight:  Default.BorderRight,
		BorderBottom: Default.BorderBottom,

		HeaderLine:      Default.HeaderLine,
		HeaderAlignment: Default.HeaderAlignment,

		RowLine:         Default.RowLine,
		ColumnSeparator: Default.ColumnSeparator,
		NewLine:         Default.NewLine,
		CenterSeparator: Default.CenterSeparator,

		DefaultAlignment: Default.DefaultAlignment,
		NumbersAlignment: Default.NumbersAlignment,

		RowLengthTitle: Default.RowLengthTitle,
		AllowRowsOnly:  Default.AllowRowsOnly,
	}
}

func (p *Printer) acquireTable() *tablewriter.Table {
	table := p.table
	if table == nil {
		table = tablewriter.NewWriter(p.out)

		// these properties can change until first `Print/Render` call.
		table.SetAlignment(int(p.DefaultAlignment))
		table.SetAutoFormatHeaders(p.AutoFormatHeaders)
		table.SetAutoWrapText(p.AutoWrapText)
		table.SetBorders(tablewriter.Border{Top: p.BorderTop, Left: p.BorderLeft, Right: p.BorderRight, Bottom: p.BorderBottom})
		table.SetHeaderLine(p.HeaderLine)
		table.SetHeaderAlignment(int(p.HeaderAlignment))
		table.SetRowLine(p.RowLine)
		table.SetColumnSeparator(p.ColumnSeparator)
		table.SetNewLine(p.NewLine)
		table.SetCenterSeparator(p.CenterSeparator)

		p.table = table
	}

	return table
}

func (p *Printer) calculateColumnAlignment(numbersColsPosition []int, size int) []int {
	columnAlignment := make([]int, size, size)
	for i := range columnAlignment {
		columnAlignment[i] = int(p.DefaultAlignment)

		for _, j := range numbersColsPosition {
			if i == j {
				columnAlignment[i] = int(p.NumbersAlignment)
				break
			}
		}
	}

	return columnAlignment
}

// Render prints a table based on the rules of this "p" Printer.
//
// It's used to customize manually the parts of a table like the headers.
// If need to append a row after its creation you should create a new `Printer` instance by calling the `New` function
// and use its `RenderRow` instead, because the "w" writer is different on each package-level printer function.
//
// Returns the total amount of rows written to the table.
func Render(w io.Writer, headers []string, rows [][]string, numbersColsPosition []int, reset bool) int {
	return New(w).Render(headers, rows, numbersColsPosition, reset)
}

// Render prints a table based on the rules of this "p" Printer.
//
// It's used to customize manually the parts of a table like the headers.
// It can be used side by side with the `RenderRow`, first and once `Render`, after and maybe many `RenderRow`.
//
// Returns the total amount of rows written to the table.
func (p *Printer) Render(headers []string, rows [][]string, numbersColsPosition []int, reset bool) int {
	table := p.acquireTable()

	if reset {
		// ClearHeaders added on kataras/tablewriter version, Changes from the original repository:
		// https://github.com/olekukonko/tablewriter/compare/master...kataras:master
		table.ClearHeaders()
		table.ClearRows()
	}

	if len(headers) > 0 {
		if p.RowLengthTitle != nil && p.RowLengthTitle(len(rows)) {
			headers[0] = fmt.Sprintf("%s (%d) ", headers[0], len(rows))
		}

		table.SetHeader(headers)
	} else if !p.AllowRowsOnly {
		return 0 // if not allow to print anything without headers, then exit.
	}

	table.AppendBulk(rows)
	table.SetColumnAlignment(p.calculateColumnAlignment(numbersColsPosition, len(headers)))

	table.Render()
	return table.NumLines()
}

// RenderRow prints a row based on the same alignment rules to the last `Print` or `Render`.
// It can be used to live update the table.
//
// Returns the total amount of rows written to the table.
func (p *Printer) RenderRow(row []string, numbersColsPosition []int) int {
	table := p.acquireTable()
	table.SetColumnAlignment(p.calculateColumnAlignment(numbersColsPosition, len(row)))

	// RenderRowOnce added on kataras/tablewriter version, Changes from the original repository:
	// https://github.com/olekukonko/tablewriter/compare/master...kataras:master
	return table.RenderRowOnce(row)
}

// Print outputs whatever "in" value passed as a table to the "w",
// filters cna be used to control what rows can be visible or hidden.
// Usage:
// Print(os.Stdout, values, func(t MyStruct) bool { /* or any type, depends on the type(s) of the "tt" */
// 	return t.Visibility != "hidden"
// })
//
// Returns the total amount of rows written to the table.
func Print(w io.Writer, v interface{}, filters ...interface{}) int {
	return New(w).Print(v, filters...)
}

// Print outputs whatever "in" value passed as a table, filters can be used to control what rows can be visible and which not.
// Usage:
// Print(values, func(t MyStruct) bool { /* or any type, depends on the type(s) of the "tt" */
// 	return t.Visibility != "hidden"
// })
//
// Returns the total amount of rows written to the table.
func (p *Printer) Print(in interface{}, filters ...interface{}) int {
	v := indirectValue(reflect.ValueOf(in))
	f := MakeFilters(v, filters...)

	headers, rows, nums := whichParser(v.Type()).Parse(v, f)

	return p.Render(headers, rows, nums, true)
}

// PrintJSON prints the json-bytes as a table to the "w",
// filters cna be used to control what rows can be visible or hidden.
//
// Returns the total amount of rows written to the table.
func PrintJSON(w io.Writer, in []byte, filters ...interface{}) int {
	return New(w).PrintJSON(in, filters...)
}

// PrintJSON prints the json-bytes as a table,
// filters cna be used to control what rows can be visible or hidden.
//
// Returns the total amount of rows written to the table.
func (p *Printer) PrintJSON(in interface{}, filters ...interface{}) int {
	v := indirectValue(reflect.ValueOf(in))
	f := MakeFilters(v, filters...)

	headers, rows, nums := JSONParser.Parse(v, f)

	return p.Render(headers, rows, nums, true)
}

// PrintHeadList prints whatever "list" as a table to the "w" with a single header.
// The "list" should be a slice of something, however
// that list can also contain different type of values, even interface{}, the function will parse each of its elements differently if needed.
//
// It can be used when want to print a simple list of string, i.e names []string, a single column each time.
//
// Returns the total amount of rows written to the table.
func PrintHeadList(w io.Writer, list interface{}, header string, filters ...interface{}) int {
	return New(w).PrintHeadList(list, header, filters...)
}

var emptyHeader StructHeader

// PrintHeadList prints whatever "list" as a table with a single header.
// The "list" should be a slice of something, however
// that list can also contain different type of values, even interface{}, the function will parse each of its elements differently if needed.
//
// It can be used when want to print a simple list of string, i.e names []string, a single column each time.
//
// Returns the total amount of rows written to the table.
func (p *Printer) PrintHeadList(list interface{}, header string, filters ...interface{}) int {
	items := indirectValue(reflect.ValueOf(list))
	if items.Kind() != reflect.Slice {
		return 0
	}

	var (
		rows                [][]string
		numbersColsPosition []int
	)

	for i, n := 0, items.Len(); i < n; i++ {
		item := items.Index(i)
		c, r := extractCells(i, emptyHeader, indirectValue(item), true)
		rows = append(rows, r)
		numbersColsPosition = append(numbersColsPosition, c...)
	}

	headers := []string{header}
	return p.Render(headers, rows, numbersColsPosition, true)
}
