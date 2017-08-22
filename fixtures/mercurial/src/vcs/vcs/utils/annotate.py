import StringIO

from pygments.formatters import HtmlFormatter
from pygments import highlight

from vcs.exceptions import VCSError
from vcs.nodes import FileNode


def annotate_highlight(filenode, annotate_from_changeset_func=None,
        order=None, headers=None, **options):
    """
    Returns html portion containing annotated table with 3 columns: line
    numbers, changeset information and pygmentized line of code.

    :param filenode: FileNode object
    :param annotate_from_changeset_func: function taking changeset and
      returning single annotate cell; needs break line at the end
    :param order: ordered sequence of ``ls`` (line numbers column),
      ``annotate`` (annotate column), ``code`` (code column); Default is
      ``['ls', 'annotate', 'code']``
    :param headers: dictionary with headers (keys are whats in ``order``
      parameter)
    """
    options['linenos'] = True
    formatter = AnnotateHtmlFormatter(filenode=filenode, order=order,
        headers=headers,
        annotate_from_changeset_func=annotate_from_changeset_func, **options)
    lexer = filenode.lexer
    highlighted = highlight(filenode.content, lexer, formatter)
    return highlighted


class AnnotateHtmlFormatter(HtmlFormatter):

    def __init__(self, filenode, annotate_from_changeset_func=None,
            order=None, **options):
        """
        If ``annotate_from_changeset_func`` is passed it should be a function
        which returns string from the given changeset. For example, we may pass
        following function as ``annotate_from_changeset_func``::

            def changeset_to_anchor(changeset):
                return '<a href="/changesets/%s/">%s</a>\n' %\
                       (changeset.id, changeset.id)

        :param annotate_from_changeset_func: see above
        :param order: (default: ``['ls', 'annotate', 'code']``); order of
          columns;
        :param options: standard pygment's HtmlFormatter options, there is
          extra option tough, ``headers``. For instance we can pass::

             formatter = AnnotateHtmlFormatter(filenode, headers={
                'ls': '#',
                'annotate': 'Annotate',
                'code': 'Code',
             })

        """
        super(AnnotateHtmlFormatter, self).__init__(**options)
        self.annotate_from_changeset_func = annotate_from_changeset_func
        self.order = order or ('ls', 'annotate', 'code')
        headers = options.pop('headers', None)
        if headers and not ('ls' in headers and 'annotate' in headers and
            'code' in headers):
            raise ValueError("If headers option dict is specified it must "
                "all 'ls', 'annotate' and 'code' keys")
        self.headers = headers
        if isinstance(filenode, FileNode):
            self.filenode = filenode
        else:
            raise VCSError("This formatter expect FileNode parameter, not %r"
                % type(filenode))

    def annotate_from_changeset(self, changeset):
        """
        Returns full html line for single changeset per annotated line.
        """
        if self.annotate_from_changeset_func:
            return self.annotate_from_changeset_func(changeset)
        else:
            return ''.join((changeset.id, '\n'))

    def _wrap_tablelinenos(self, inner):
        dummyoutfile = StringIO.StringIO()
        lncount = 0
        for t, line in inner:
            if t:
                lncount += 1
            dummyoutfile.write(line)

        fl = self.linenostart
        mw = len(str(lncount + fl - 1))
        sp = self.linenospecial
        st = self.linenostep
        la = self.lineanchors
        aln = self.anchorlinenos
        if sp:
            lines = []

            for i in range(fl, fl + lncount):
                if i % st == 0:
                    if i % sp == 0:
                        if aln:
                            lines.append('<a href="#%s-%d" class="special">'
                                         '%*d</a>' %
                                         (la, i, mw, i))
                        else:
                            lines.append('<span class="special">'
                                         '%*d</span>' % (mw, i))
                    else:
                        if aln:
                            lines.append('<a href="#%s-%d">'
                                         '%*d</a>' % (la, i, mw, i))
                        else:
                            lines.append('%*d' % (mw, i))
                else:
                    lines.append('')
            ls = '\n'.join(lines)
        else:
            lines = []
            for i in range(fl, fl + lncount):
                if i % st == 0:
                    if aln:
                        lines.append('<a href="#%s-%d">%*d</a>' \
                                     % (la, i, mw, i))
                    else:
                        lines.append('%*d' % (mw, i))
                else:
                    lines.append('')
            ls = '\n'.join(lines)

        annotate_changesets = [tup[1] for tup in self.filenode.annotate]
        # If pygments cropped last lines break we need do that too
        ln_cs = len(annotate_changesets)
        ln_ = len(ls.splitlines())
        if  ln_cs > ln_:
            annotate_changesets = annotate_changesets[:ln_ - ln_cs]
        annotate = ''.join((self.annotate_from_changeset(changeset)
            for changeset in annotate_changesets))
        # in case you wonder about the seemingly redundant <div> here:
        # since the content in the other cell also is wrapped in a div,
        # some browsers in some configurations seem to mess up the formatting.
        '''
        yield 0, ('<table class="%stable">' % self.cssclass +
                  '<tr><td class="linenos"><div class="linenodiv"><pre>' +
                  ls + '</pre></div></td>' +
                  '<td class="code">')
        yield 0, dummyoutfile.getvalue()
        yield 0, '</td></tr></table>'

        '''
        headers_row = []
        if self.headers:
            headers_row = ['<tr class="annotate-header">']
            for key in self.order:
                td = ''.join(('<td>', self.headers[key], '</td>'))
                headers_row.append(td)
            headers_row.append('</tr>')

        body_row_start = ['<tr>']
        for key in self.order:
            if key == 'ls':
                body_row_start.append(
                    '<td class="linenos"><div class="linenodiv"><pre>' +
                    ls + '</pre></div></td>')
            elif key == 'annotate':
                body_row_start.append(
                    '<td class="annotate"><div class="annotatediv"><pre>' +
                    annotate + '</pre></div></td>')
            elif key == 'code':
                body_row_start.append('<td class="code">')
        yield 0, ('<table class="%stable">' % self.cssclass +
                  ''.join(headers_row) +
                  ''.join(body_row_start)
                  )
        yield 0, dummyoutfile.getvalue()
        yield 0, '</td></tr></table>'
