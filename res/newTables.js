

tableApp = new Vue({
    el: '#rightTable',
    data() {
        return {
            message: 'Hello Vue!',
            tables: [],
            isShow: false,
            tableNameFilterText: '',
            selectMode: false
        }
    },
    computed: {
        switchText() {
            return this.isShow ? '合' : '开'
        },
        filterTables() {
            let parts = this.tableNameFilterText.split('|')
                .map(item => item.trim())
                .filter(item => item.length > 0)
                .map(item => item.toUpperCase())
            if (parts.length === 0) return this.tables;

            return this.tables.filter(t => parts.some(part => t.name.toUpperCase().indexOf(part) > -1))
        }
    },
    methods: {
        initTable() {
            var self = this
            var tid = activeMerchantId
            var withColumns = !$.withColumnsCache[tid]
            $.ajax({
                type: 'POST',
                url: contextPath + "/query",
                data: {tid: tid, sql: 'show tables', withColumns: withColumns},
                success: function (content, textStatus, request) {
                    if (content && content.Error) {
                        $.alertMe(content.Error)
                        return
                    }

                    if (withColumns) {
                        $.withColumnsCache[tid] = content.TableColumns
                    }
                    self.showTables(content, $.createTableColumns(tid))
                },
                error: function (jqXHR, textStatus, errorThrown) {
                    $.alertMe(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
                }
            })
        },
        showTables(content, tableColumns) {
            this.tables = content.Rows.map(row => {
                return {name: row[1], checked: false}
            })
            this.isShow = true
            $('.MainDivs').addClass('MainDivsShowTable')

            var timeout = null
            var editor = $.sqlCodeMirror
            editor.off("keyup")
            editor.on("keyup", function (cm, event) {
                if ($.ExcludedIntelliSenseTriggerKeys[(event.keyCode || event.which).toString()]
                    || cm.state.completionActive) return

                var cur = cm.getCursor()
                var ch = cm.getRange(CodeMirror.Pos(cur.line, cur.ch - 1), cur)
                if (!$.isCharForShowHint(ch)) return

                var tok = cm.getTokenAt(cur)
                if (tok.type === "string" && tok.string.length >= 1 && tok.string.substr(0, 1) === "'") return false;

                if (timeout) clearTimeout(timeout)
                timeout = setTimeout(function () {
                    CodeMirror.showHint(cm, CodeMirror.hint.sql, {
                        // "completeSingle: false" prevents case when you are typing some word
                        // and in the middle it is automatically completed and you continue typing by reflex.
                        // So user will always need to select the intended string
                        // from popup (even if it's single option). (copy from @Oleksandr Pshenychnyy)
                        completeSingle: false,
                        tables: tableColumns
                    })
                }, 150)
            })
            $.contextMenu({
                zIndex: 10,
                selector: '.itemSpan',
                callback: function (key, options) {
                    var tableName = $(this).text()
                    if (key === 'ShowFullColumns') {
                        $.executeQueryAjax(activeClassifier, activeMerchantId, activeMerchantCode, activeMerchantName, 'show full columns from ' + tableName)
                    } else if (key === 'ShowCreateTable') {
                        $.showSqlAjax('show create table ' + tableName)
                    } else if (key === 'RenameTable') {
                        $.appendSqlToSqlEditor('RENAME TABLE ' + tableName + ' TO ' + tableName + "_new", true, false)
                    }
                },
                items: {
                    ShowFullColumns: {name: 'Show Columns', icon: 'columns'},
                    ShowCreateTable: {name: 'Show Create Table', icon: 'create-table'},
                    RenameTable: {name: 'Rename Table', icon: 'create-table'},
                }
            })
        },
        switchShow() {
            this.isShow = !this.isShow
            $('.MainDivs').toggleClass('MainDivsShowTable', this.isShow)
        }
        ,
        selectTable(tableName) {
            if (this.selectMode) return

            var sql = 'select * from ' + tableName
            var tableQuery = $.SingleTableQuery[tableName.toUpperCase()];

            if (tableQuery) {
                if (tableQuery.replaceSql) sql = tableQuery.replaceSql
                if (tableQuery.appendSql) sql += ' ' + tableQuery.appendSql
            }

            $.executeQueryAjax(activeClassifier, activeMerchantId, activeMerchantCode, activeMerchantName, sql)
        },
        checkedTables() {
            return this.filterTables.filter(t=>t.checked).map(t=>t.name)
        },
        renameTables() {
            const ts = this.checkedTables()
            if (ts.length <= 0) return

            let tables = ts.map((table) => table + ' to ' + table + '_{{new}}').join(', ')
            $.appendSqlToSqlEditor('RENAME TABLE ' + tables, true, false)
        },
        truncateTables() {
            const ts = this.checkedTables()
            if (ts.length <= 0) return

            $.executeMultiSqlsAjax(`truncate table ${ts.join(';\ntruncate table ')};`, true)
        },
        dumpTables() {
            const ts = this.checkedTables()
            if (ts.length <= 0) return

            $.exportDbImpl(activeMerchantId, activeMerchantCode, activeHomeArea, activeClassifier, activeMerchantName,
                `${ts.join(',')}`)
        }
    }
})

