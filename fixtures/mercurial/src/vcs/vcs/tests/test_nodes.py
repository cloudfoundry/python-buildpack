from __future__ import with_statement

import stat
from vcs.nodes import DirNode
from vcs.nodes import FileNode
from vcs.nodes import Node
from vcs.nodes import NodeError
from vcs.nodes import NodeKind
from vcs.utils.compat import unittest


class NodeBasicTest(unittest.TestCase):

    def test_init(self):
        """
        Cannot innitialize Node objects with path with slash at the beginning.
        """
        wrong_paths = (
            '/foo',
            '/foo/bar'
        )
        for path in wrong_paths:
            self.assertRaises(NodeError, Node, path, NodeKind.FILE)

        wrong_paths = (
            '/foo/',
            '/foo/bar/'
        )
        for path in wrong_paths:
            self.assertRaises(NodeError, Node, path, NodeKind.DIR)

    def test_name(self):
        node = Node('', NodeKind.DIR)
        self.assertEqual(node.name, '')

        node = Node('path', NodeKind.FILE)
        self.assertEqual(node.name, 'path')

        node = Node('path/', NodeKind.DIR)
        self.assertEqual(node.name, 'path')

        node = Node('some/path', NodeKind.FILE)
        self.assertEqual(node.name, 'path')

        node = Node('some/path/', NodeKind.DIR)
        self.assertEqual(node.name, 'path')

    def test_root_node(self):
        self.assertRaises(NodeError, Node, '', NodeKind.FILE)

    def test_kind_setter(self):
        node = Node('', NodeKind.DIR)
        self.assertRaises(NodeError, setattr, node, 'kind', NodeKind.FILE)

    def _test_parent_path(self, node_path, expected_parent_path):
        """
        Tests if node's parent path are properly computed.
        """
        node = Node(node_path, NodeKind.DIR)
        parent_path = node.get_parent_path()
        self.assertTrue(parent_path.endswith('/') or \
            node.is_root() and parent_path == '')
        self.assertEqual(parent_path, expected_parent_path,
            "Node's path is %r and parent path is %r but should be %r"
            % (node.path, parent_path, expected_parent_path))

    def test_parent_path(self):
        test_paths = (
            # (node_path, expected_parent_path)
            ('', ''),
            ('some/path/', 'some/'),
            ('some/longer/path/', 'some/longer/'),
        )
        for node_path, expected_parent_path in test_paths:
            self._test_parent_path(node_path, expected_parent_path)

    '''
    def _test_trailing_slash(self, path):
        if not path.endswith('/'):
            self.fail("Trailing slash tests needs paths to end with slash")
        for kind in NodeKind.FILE, NodeKind.DIR:
            self.assertRaises(NodeError, Node, path=path, kind=kind)

    def test_trailing_slash(self):
        for path in ('/', 'foo/', 'foo/bar/', 'foo/bar/biz/'):
            self._test_trailing_slash(path)
    '''

    def test_is_file(self):
        node = Node('any', NodeKind.FILE)
        self.assertTrue(node.is_file())

        node = FileNode('any')
        self.assertTrue(node.is_file())
        self.assertRaises(AttributeError, getattr, node, 'nodes')

    def test_is_dir(self):
        node = Node('any_dir', NodeKind.DIR)
        self.assertTrue(node.is_dir())

        node = DirNode('any_dir')

        self.assertTrue(node.is_dir())
        self.assertRaises(NodeError, getattr, node, 'content')

    def test_dir_node_iter(self):
        nodes = [
            DirNode('docs'),
            DirNode('tests'),
            FileNode('bar'),
            FileNode('foo'),
            FileNode('readme.txt'),
            FileNode('setup.py'),
        ]
        dirnode = DirNode('', nodes=nodes)
        for node in dirnode:
            node == dirnode.get_node(node.path)

    def test_node_state(self):
        """
        Without link to changeset nodes should raise NodeError.
        """
        node = FileNode('anything')
        self.assertRaises(NodeError, getattr, node, 'state')
        node = DirNode('anything')
        self.assertRaises(NodeError, getattr, node, 'state')

    def test_file_node_stat(self):
        node = FileNode('foobar', 'empty... almost')
        mode = node.mode  # default should be 0100644
        self.assertTrue(mode & stat.S_IRUSR)
        self.assertTrue(mode & stat.S_IWUSR)
        self.assertTrue(mode & stat.S_IRGRP)
        self.assertTrue(mode & stat.S_IROTH)
        self.assertFalse(mode & stat.S_IWGRP)
        self.assertFalse(mode & stat.S_IWOTH)
        self.assertFalse(mode & stat.S_IXUSR)
        self.assertFalse(mode & stat.S_IXGRP)
        self.assertFalse(mode & stat.S_IXOTH)

    def test_file_node_is_executable(self):
        node = FileNode('foobar', 'empty... almost', mode=0100755)
        self.assertTrue(node.is_executable())

        node = FileNode('foobar', 'empty... almost', mode=0100500)
        self.assertTrue(node.is_executable())

        node = FileNode('foobar', 'empty... almost', mode=0100644)
        self.assertFalse(node.is_executable())

    def test_mimetype(self):
        py_node = FileNode('test.py')
        tar_node = FileNode('test.tar.gz')

        ext = 'CustomExtension'

        my_node2 = FileNode('myfile2')
        my_node2._mimetype = [ext]

        my_node3 = FileNode('myfile3')
        my_node3._mimetype = [ext,ext]

        self.assertEqual(py_node.mimetype,'text/x-python')
        self.assertEqual(py_node.get_mimetype(),('text/x-python',None))

        self.assertEqual(tar_node.mimetype,'application/x-tar')
        self.assertEqual(tar_node.get_mimetype(),('application/x-tar','gzip'))

        self.assertRaises(NodeError,my_node2.get_mimetype)

        self.assertEqual(my_node3.mimetype,ext)
        self.assertEqual(my_node3.get_mimetype(),[ext,ext])

class NodeContentTest(unittest.TestCase):

    def test_if_binary(self):
        data = """\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10\x00\x00\x00\x10\x08\x06\x00\x00\x00\x1f??a\x00\x00\x00\x04gAMA\x00\x00\xaf?7\x05\x8a?\x00\x00\x00\x19tEXtSoftware\x00Adobe ImageReadyq?e<\x00\x00\x025IDAT8?\xa5\x93?K\x94Q\x14\x87\x9f\xf7?Q\x1bs4?\x03\x9a\xa8?B\x02\x8b$\x10[U;i\x13?6h?&h[?"\x14j?\xa2M\x7fB\x14F\x9aQ?&\x842?\x0b\x89"\x82??!?\x9c!\x9c2l??{N\x8bW\x9dY\xb4\t/\x1c?=\x9b?}????\xa9*;9!?\x83\x91?[?\\v*?D\x04\'`EpNp\xa2X\'U?pVq"Sw.\x1e?\x08\x01D?jw????\xbc??7{|\x9b?\x89$\x01??W@\x15\x9c\x05q`Lt/\x97?\x94\xa1d?\x18~?\x18?\x18W[%\xb0?\x83??\x14\x88\x8dB?\xa6H\tL\tl\x19>/\x01`\xac\xabx?\x9cl\nx\xb0\x98\x07\x95\x88D$"q[\x19?d\x00(o\n\xa0??\x7f\xb9\xa4?\x1bF\x1f\x8e\xac\xa8?j??eUU}?.?\x9f\x8cE??x\x94??\r\xbdtoJU5"0N\x10U?\x00??V\t\x02\x9f\x81?U?\x00\x9eM\xae2?r\x9b7\x83\x82\x8aP3????.?&"?\xb7ZP \x0c<?O\xa5\t}\xb8?\x99\xa6?\x87?\x1di|/\xa0??0\xbe\x1fp?d&\x1a\xad\x95\x8a\x07?\t*\x10??b:?d?.\x13C\x8a?\x12\xbe\xbf\x8e?{???\x08?\x80\xa7\x13+d\x13>J?\x80\x15T\x95\x9a\x00??S\x8c\r?\xa1\x03\x07?\x96\x9b\xa7\xab=E??\xa4\xb3?\x19q??B\x91=\x8d??k?J\x0bV"??\xf7x?\xa1\x00?\\.\x87\x87???\x02F@D\x99],??\x10#?X\xb7=\xb9\x10?Z\x1by???cI??\x1ag?\x92\xbc?T?t[\x92\x81?<_\x17~\x92\x88?H%?\x10Q\x02\x9f\n\x81qQ\x0bm?\x1bX?\xb1AK\xa6\x9e\xb9?u\xb2?1\xbe|/\x92M@\xa2!F?\xa9>"\r<DT?>\x92\x8e?>\x9a9Qv\x127?a\xac?Y?8?:??]X???9\x80\xb7?u?\x0b#BZ\x8d=\x1d?p\x00\x00\x00\x00IEND\xaeB`\x82"""
        filenode = FileNode('calendar.png', content=data)
        self.assertTrue(filenode.is_binary)


if __name__ == '__main__':
    unittest.main()
