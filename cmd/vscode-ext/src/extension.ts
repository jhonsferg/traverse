import * as vscode from 'vscode';

const SYSTEM_QUERY_OPTIONS: vscode.CompletionItem[] = [
    mkKeyword('$filter', 'Filters the collection. `$filter=Price gt 100`'),
    mkKeyword('$select', 'Selects specific properties. `$select=ID,Name,Price`'),
    mkKeyword('$expand', 'Expands related entities. `$expand=Orders($select=ID)`'),
    mkKeyword('$orderby', 'Sorts results. `$orderby=Name asc,Price desc`'),
    mkKeyword('$top', 'Limits number of results. `$top=50`'),
    mkKeyword('$skip', 'Skips N results for pagination. `$skip=100`'),
    mkKeyword('$count', 'Includes total count. `$count=true`'),
    mkKeyword('$search', 'Full-text search. `$search=laptop`'),
    mkKeyword('$apply', 'Aggregation pipeline. `$apply=groupby((Category),aggregate(Price with sum as Total))`'),
    mkKeyword('$compute', 'Computed properties. `$compute=Price mul Qty as LineTotal`'),
    mkKeyword('$format', 'Response format. `$format=json`'),
    mkKeyword('$skiptoken', 'Server-side pagination token.'),
    mkKeyword('$deltatoken', 'Delta synchronisation token.'),
];

const LOGICAL_OPERATORS: vscode.CompletionItem[] = [
    mkOperator('eq', 'Equal — `Field eq Value`'),
    mkOperator('ne', 'Not equal — `Field ne Value`'),
    mkOperator('lt', 'Less than — `Field lt Value`'),
    mkOperator('le', 'Less than or equal — `Field le Value`'),
    mkOperator('gt', 'Greater than — `Field gt Value`'),
    mkOperator('ge', 'Greater than or equal — `Field ge Value`'),
    mkOperator('and', 'Logical AND — `A eq 1 and B eq 2`'),
    mkOperator('or', 'Logical OR — `A eq 1 or B eq 2`'),
    mkOperator('not', 'Logical NOT — `not (A eq 1)`'),
    mkOperator('in', 'In list — `Status in (\'A\',\'B\')`'),
    mkOperator('has', 'Has enum flag — `Style has Sales.Pattern\'Yellow\'`'),
    mkOperator('add', 'Arithmetic add'),
    mkOperator('sub', 'Arithmetic subtract'),
    mkOperator('mul', 'Arithmetic multiply'),
    mkOperator('div', 'Arithmetic divide (integer)'),
    mkOperator('divby', 'Arithmetic divide (decimal)'),
    mkOperator('mod', 'Arithmetic modulo'),
    mkOperator('asc', 'Ascending order for $orderby'),
    mkOperator('desc', 'Descending order for $orderby'),
];

const FUNCTIONS: { label: string; sig: string; doc: string }[] = [
    { label: 'contains',        sig: 'contains(field, value)',              doc: 'Returns true if the field contains the substring.\n\n`$filter=contains(Name, \'SAP\')`' },
    { label: 'startswith',      sig: 'startswith(field, value)',            doc: 'Returns true if the field starts with the substring.\n\n`$filter=startswith(Name, \'A\')`' },
    { label: 'endswith',        sig: 'endswith(field, value)',              doc: 'Returns true if the field ends with the substring.\n\n`$filter=endswith(Email, \'.com\')`' },
    { label: 'length',          sig: 'length(field)',                       doc: 'Returns the length of a string property.\n\n`$filter=length(Name) gt 5`' },
    { label: 'indexof',         sig: 'indexof(field, value)',               doc: 'Returns the zero-based index of the first occurrence.\n\n`$filter=indexof(Name, \'SAP\') ge 0`' },
    { label: 'substring',       sig: 'substring(field, start[, length])',   doc: 'Returns a substring.\n\n`$filter=substring(Name,1,3) eq \'APE\'`' },
    { label: 'tolower',         sig: 'tolower(field)',                      doc: 'Converts to lower-case.\n\n`$filter=tolower(Name) eq \'sap\'`' },
    { label: 'toupper',         sig: 'toupper(field)',                      doc: 'Converts to upper-case.\n\n`$filter=toupper(Name) eq \'SAP\'`' },
    { label: 'trim',            sig: 'trim(field)',                         doc: 'Removes leading and trailing whitespace.\n\n`$filter=trim(Name) eq \'SAP\'`' },
    { label: 'concat',          sig: 'concat(a, b)',                        doc: 'Concatenates two string values.\n\n`$filter=concat(FirstName,LastName) eq \'JohnDoe\'`' },
    { label: 'matchesPattern',  sig: 'matchesPattern(field, pattern)',      doc: 'OData 4.0 regex match.\n\n`$filter=matchesPattern(Name, \'S.*\')`' },
    { label: 'substringof',     sig: 'substringof(value, field)',           doc: 'OData v2 contains equivalent (note: arguments reversed).\n\n`$filter=substringof(\'SAP\', Name)`' },
    { label: 'year',            sig: 'year(field)',                         doc: 'Extracts year from a date/datetime property.\n\n`$filter=year(CreatedAt) eq 2026`' },
    { label: 'month',           sig: 'month(field)',                        doc: 'Extracts month (1–12).\n\n`$filter=month(CreatedAt) eq 4`' },
    { label: 'day',             sig: 'day(field)',                          doc: 'Extracts day of month (1–31).\n\n`$filter=day(CreatedAt) eq 7`' },
    { label: 'hour',            sig: 'hour(field)',                         doc: 'Extracts hour (0–23).' },
    { label: 'minute',          sig: 'minute(field)',                       doc: 'Extracts minute (0–59).' },
    { label: 'second',          sig: 'second(field)',                       doc: 'Extracts second (0–59).' },
    { label: 'now',             sig: 'now()',                               doc: 'Current date and time.\n\n`$filter=CreatedAt gt now()`' },
    { label: 'mindatetime',     sig: 'mindatetime()',                       doc: 'Minimum possible DateTimeOffset value.' },
    { label: 'maxdatetime',     sig: 'maxdatetime()',                       doc: 'Maximum possible DateTimeOffset value.' },
    { label: 'totaloffsetminutes', sig: 'totaloffsetminutes(field)',        doc: 'Returns total offset in minutes from UTC.' },
    { label: 'date',            sig: 'date(field)',                         doc: 'Extracts the date part from a DateTimeOffset.' },
    { label: 'time',            sig: 'time(field)',                         doc: 'Extracts the time part from a DateTimeOffset.' },
    { label: 'totalseconds',    sig: 'totalseconds(field)',                 doc: 'Total duration in seconds.' },
    { label: 'fractionalseconds', sig: 'fractionalseconds(field)',          doc: 'Fractional part of seconds.' },
    { label: 'round',           sig: 'round(field)',                        doc: 'Rounds to the nearest integer.\n\n`$filter=round(Price) eq 10`' },
    { label: 'floor',           sig: 'floor(field)',                        doc: 'Rounds down to integer.\n\n`$filter=floor(Price) eq 9`' },
    { label: 'ceiling',         sig: 'ceiling(field)',                      doc: 'Rounds up to integer.\n\n`$filter=ceiling(Price) eq 10`' },
    { label: 'isof',            sig: 'isof([field,] TypeName)',             doc: 'Type check — returns true if the entity/property is of the given type.\n\n`$filter=isof(Model.Manager)`' },
    { label: 'cast',            sig: 'cast([field,] TypeName)',             doc: 'Type cast.\n\n`$filter=cast(Budget,Edm.Decimal) gt 1000`' },
    { label: 'any',             sig: 'Collection/any(x: x/Field op value)', doc: 'Lambda — returns true if any element matches.\n\n`$filter=Tags/any(t: t/Name eq \'admin\')`' },
    { label: 'all',             sig: 'Collection/all(x: x/Field op value)', doc: 'Lambda — returns true if all elements match.\n\n`$filter=Items/all(i: i/Qty gt 0)`' },
    { label: 'geo.distance',    sig: 'geo.distance(field, point)',          doc: 'Distance between two geography points (in meters).\n\n`$filter=geo.distance(Location, geography\'SRID=4326;POINT(-122.1 47.6)\') lt 10000`' },
    { label: 'geo.intersects',  sig: 'geo.intersects(point, polygon)',      doc: 'Returns true if the point is within the polygon.\n\n`$filter=geo.intersects(Location, geography\'SRID=4326;POLYGON(...)\')`' },
    { label: 'geo.length',      sig: 'geo.length(linestring)',              doc: 'Returns the length of the linestring in meters.' },
];

const EDM_TYPES: vscode.CompletionItem[] = [
    'Edm.String', 'Edm.Int16', 'Edm.Int32', 'Edm.Int64',
    'Edm.Decimal', 'Edm.Double', 'Edm.Single', 'Edm.Boolean',
    'Edm.Guid', 'Edm.DateTime', 'Edm.DateTimeOffset', 'Edm.Date',
    'Edm.TimeOfDay', 'Edm.Duration', 'Edm.Binary', 'Edm.Stream',
    'Edm.GeographyPoint', 'Edm.GeographyLineString', 'Edm.GeographyPolygon',
    'Edm.GeometryPoint', 'Edm.GeometryLineString', 'Edm.GeometryPolygon',
].map(t => {
    const item = new vscode.CompletionItem(t, vscode.CompletionItemKind.TypeParameter);
    item.detail = 'OData EDM Type';
    return item;
});

function mkKeyword(label: string, doc: string): vscode.CompletionItem {
    const item = new vscode.CompletionItem(label, vscode.CompletionItemKind.Keyword);
    item.documentation = new vscode.MarkdownString(doc);
    item.detail = 'OData system query option';
    return item;
}

function mkOperator(label: string, doc: string): vscode.CompletionItem {
    const item = new vscode.CompletionItem(label, vscode.CompletionItemKind.Operator);
    item.documentation = new vscode.MarkdownString(doc);
    item.detail = 'OData operator';
    return item;
}

function makeFunctionCompletions(): vscode.CompletionItem[] {
    return FUNCTIONS.map(fn => {
        const item = new vscode.CompletionItem(fn.label, vscode.CompletionItemKind.Function);
        item.detail = fn.sig;
        item.documentation = new vscode.MarkdownString(fn.doc);
        item.insertText = new vscode.SnippetString(`${fn.label}($1)`);
        return item;
    });
}

const FUNCTION_COMPLETIONS = makeFunctionCompletions();
const ALL_COMPLETIONS = [...SYSTEM_QUERY_OPTIONS, ...LOGICAL_OPERATORS, ...FUNCTION_COMPLETIONS, ...EDM_TYPES];

const HOVER_DOCS: Map<string, string> = new Map(
    FUNCTIONS.map(fn => [fn.label, `**\`${fn.sig}\`**\n\n${fn.doc}`])
);

for (const op of LOGICAL_OPERATORS) {
    if (op.documentation instanceof vscode.MarkdownString) {
        HOVER_DOCS.set(op.label as string, op.documentation.value);
    }
}

function isInsideODataString(document: vscode.TextDocument, position: vscode.Position): boolean {
    const lineText = document.lineAt(position).text.substring(0, position.character);
    return /\.(Filter|Select|Expand|OrderBy|Search|Apply|Compute)\s*\(["']/.test(lineText) ||
           /\.(Where|Having)\s*\(["']/.test(lineText) ||
           document.languageId === 'odata';
}

function getWordAtPosition(document: vscode.TextDocument, position: vscode.Position): string {
    const range = document.getWordRangeAtPosition(position, /[\w$.]+/);
    return range ? document.getText(range) : '';
}

class ODataCompletionProvider implements vscode.CompletionItemProvider {
    provideCompletionItems(
        document: vscode.TextDocument,
        position: vscode.Position
    ): vscode.CompletionItem[] | undefined {
        const config = vscode.workspace.getConfiguration('traverse.odata');
        const mode = config.get<string>('completionMode', 'full');
        if (mode === 'off') { return undefined; }

        if (!isInsideODataString(document, position)) {
            return undefined;
        }

        if (mode === 'keywords-only') {
            return [...SYSTEM_QUERY_OPTIONS, ...LOGICAL_OPERATORS];
        }

        return ALL_COMPLETIONS;
    }
}

class ODataHoverProvider implements vscode.HoverProvider {
    provideHover(
        document: vscode.TextDocument,
        position: vscode.Position
    ): vscode.Hover | undefined {
        const config = vscode.workspace.getConfiguration('traverse.odata');
        if (!config.get<boolean>('hoverDocs', true)) { return undefined; }

        const word = getWordAtPosition(document, position);
        const doc = HOVER_DOCS.get(word);
        if (!doc) { return undefined; }

        return new vscode.Hover(new vscode.MarkdownString(doc));
    }
}

interface DiagnosticResult {
    message: string;
    startChar: number;
    endChar: number;
}

function validateODataFilter(expr: string): DiagnosticResult[] {
    const results: DiagnosticResult[] = [];
    let depth = 0;
    let inString = false;
    let stringStart = -1;

    for (let i = 0; i < expr.length; i++) {
        const ch = expr[i];
        if (ch === "'" && !inString) {
            inString = true;
            stringStart = i;
        } else if (ch === "'" && inString) {
            if (expr[i + 1] === "'") {
                i++;
            } else {
                inString = false;
                stringStart = -1;
            }
        } else if (!inString) {
            if (ch === '(') { depth++; }
            else if (ch === ')') {
                depth--;
                if (depth < 0) {
                    results.push({ message: 'Unmatched closing parenthesis', startChar: i, endChar: i + 1 });
                    depth = 0;
                }
            }
        }
    }

    if (inString && stringStart >= 0) {
        results.push({ message: 'Unterminated string literal', startChar: stringStart, endChar: expr.length });
    }

    if (depth > 0) {
        results.push({ message: `${depth} unclosed parenthesis(es)`, startChar: 0, endChar: expr.length });
    }

    return results;
}

const FILTER_ARG_PATTERN = /\.(Filter|Where)\s*\(\s*"([^"\\]*(\\.[^"\\]*)*)"/g;

function diagnoseDocument(document: vscode.TextDocument, collection: vscode.DiagnosticCollection): void {
    const config = vscode.workspace.getConfiguration('traverse.odata');
    if (!config.get<boolean>('validateOnType', true)) {
        collection.delete(document.uri);
        return;
    }

    const diagnostics: vscode.Diagnostic[] = [];
    const text = document.getText();

    for (const match of text.matchAll(FILTER_ARG_PATTERN)) {
        const filterExpr = match[2];
        const matchStart = match.index ?? 0;
        const argOffset = match[0].indexOf('"') + 1 + matchStart;

        const errors = validateODataFilter(filterExpr);
        for (const err of errors) {
            const startPos = document.positionAt(argOffset + err.startChar);
            const endPos   = document.positionAt(argOffset + err.endChar);
            const diag = new vscode.Diagnostic(
                new vscode.Range(startPos, endPos),
                `OData: ${err.message}`,
                vscode.DiagnosticSeverity.Warning
            );
            diag.source = 'traverse-odata';
            diagnostics.push(diag);
        }
    }

    collection.set(document.uri, diagnostics);
}

export function activate(context: vscode.ExtensionContext): void {
    const diagnosticCollection = vscode.languages.createDiagnosticCollection('traverse-odata');
    context.subscriptions.push(diagnosticCollection);

    const completionProvider = vscode.languages.registerCompletionItemProvider(
        [{ language: 'odata' }, { language: 'go' }],
        new ODataCompletionProvider(),
        '$', '(', ' ', "'"
    );
    context.subscriptions.push(completionProvider);

    const hoverProvider = vscode.languages.registerHoverProvider(
        [{ language: 'odata' }, { language: 'go' }],
        new ODataHoverProvider()
    );
    context.subscriptions.push(hoverProvider);

    const diagnose = (doc: vscode.TextDocument) => {
        if (doc.languageId === 'go' || doc.languageId === 'odata') {
            diagnoseDocument(doc, diagnosticCollection);
        }
    };

    vscode.workspace.textDocuments.forEach(diagnose);

    context.subscriptions.push(
        vscode.workspace.onDidOpenTextDocument(diagnose),
        vscode.workspace.onDidChangeTextDocument(e => diagnose(e.document)),
        vscode.workspace.onDidCloseTextDocument(doc => diagnosticCollection.delete(doc.uri))
    );
}

export function deactivate(): void {}
