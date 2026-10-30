import json
from pathlib import Path
import tempfile
import unittest
from types import SimpleNamespace

import contract_entry as entry


class ContractEntryTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.write('.ptsip/index.json', {'namespaces': {'contracts': {'status':'ACTIVE','index':'contracts/index.json'}}})
        self.write('.ptsip/contracts/index.json', {
            'policy': 'policy/boundary.json',
            'responsibility_authority': 'Responsibility-Authority/Responsibility-Authority.json',
            'release_documentation': '.ptsip/contracts/release/Platform/Common/releasenote.json',
            'release': '.ptsip/contracts/release/index.json',
            'authoring': {'entrypoint': '.ptsip/engine/developer', 'validators': ['policy-rules','responsibility-authority'], 'execution_authorized': False}
        })
        self.write('.ptsip/engine/developer/index.json', {'execution_authorized':False,'policy_authority':'NONE', 'validators': {'policy-check':'policy/index.json','responsibility-authority':'x','release-documentation':'y','release-admission-check':'z'}})
        self.write('.ptsip/contracts/policy/boundary.json', {'routing_binding': {'data':'.ptsip/contracts/policy/routing/policy_routes.json'}, 'rules_binding': {'directory': '.ptsip/contracts/policy/Rules', 'family_source': '.ptsip/contracts/policy/identity/root_family.json', 'responsibility_profile':'.ptsip/profiles/Responsibility-Authority-list.yaml'}})
        self.write('.ptsip/contracts/policy/routing/policy_routes.json', {'routes':{'PTSIP_DEVELOPER_POLICY':{'output_root':'developer/policy'}}, 'identity_sources':{'policy_kind':'.ptsip/contracts/policy/identity/policy_kind.json'}})
        self.write('.ptsip/contracts/policy/identity/policy_kind.json', {'enum':['PTSIP_DEVELOPER_POLICY']})
        self.write('.ptsip/contracts/policy/identity/root_family.json', {'enum':['GOV']})
        self.write('.ptsip/contracts/policy/Rules/GOV_rules.json', {'root_family':'GOV'})
        self.write('.ptsip/contracts/Responsibility-Authority/Responsibility-Authority.json', {})
        self.write('.ptsip/profiles/Responsibility-Authority-list.yaml', 'responsibility: []')
        self.write('developer/policy/GOV/MPD-GOV-0006.yaml', {'Policy_Identity': {'root_family': 'GOV', 'policy_kind':'PTSIP_DEVELOPER_POLICY'}, 'Policy_Status':{'status':'APPROVED.Applying'}})
        self.write('.ptsip/contracts/release/Platform/Common/releasenote.json',{'canonical_readme':{'path':'README.md'}})
        self.write('README.md','PTSIP')
        self.write('.ptsip/contracts/release/index.json', {'contract_ref':'.ptsip/contracts/release/Platform/Common/verification-boundary.json', 'validation_rules_ref':'.ptsip/contracts/release/Platform/Common/release_validation.json', 'validator_name':'release-admission-check','platforms':[{'contracts': {'verification':'.ptsip/contracts/release/Platform/Common/verification-boundary.json'},'validation_ref':'.ptsip/contracts/release/Platform/Common/Common_validation.json'}]})
        self.write('.ptsip/contracts/release/Platform/Common/verification-boundary.json', {})

    def tearDown(self):
        self.temp.cleanup()

    def write(self, path, data):
        p=self.root/path
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(json.dumps(data) if isinstance(data,(dict,list)) else data, encoding='utf8')

    def test_policy_auto_selection(self):
        item=entry.resolve(self.root, 'developer/policy/GOV/MPD-GOV-0006.yaml')
        self.assertEqual('policy-check',item['validator'])
        self.assertEqual('GOV',item['root_family'])
        self.assertEqual('check',item['argv'][-3])

    def test_non_mutating_execution(self):
        calls=[]
        def fake_runner(argv, **kw):
            calls.append((argv,kw))
            return SimpleNamespace(returncode=0,stdout='{"status":"VALID"}',stderr='')
        report,code=entry.run(self.root,'developer/policy/GOV/MPD-GOV-0006.yaml',runner=fake_runner)
        self.assertEqual(0,code)
        self.assertEqual('CHECK_COMPLETED',report['status'])
        self.assertFalse(report['mutation_authorized'])
        self.assertEqual(1,len(calls))
        self.assertIn('--validator',calls[0][0])

    def test_responsibility_and_documentation_and_release_routes(self):
        self.assertEqual('responsibility-authority',entry.resolve(self.root,'.ptsip/profiles/Responsibility-Authority-list.yaml')['validator'])
        self.assertEqual('release-documentation',entry.resolve(self.root,'README.md')['validator'])
        self.assertEqual('release-admission-check',entry.resolve(self.root,'.ptsip/contracts/release/Platform/Common/verification-boundary.json')['validator'])

    def test_reject_unregistered_and_unsupported_path(self):
        self.write('src/policy/index.yaml','hello: world')
        for target in ('src/policy/index.yaml','developer/policy/GOV/does-not-exist.yaml','../outside.yaml'):
            report,code=entry.run(self.root,target,dry_run=True)
            self.assertEqual(2,code)
            self.assertEqual('UNRESOLVED',report['status'])

    def test_reject_forged_family(self):
        self.write('developer/policy/GOV/MPD-GOV-0006.yaml',{'Policy_Identity':{'policy_kind':'PTSIP_DEVELOPER_POLICY','root_family':'UNKNOWN'}})
        report,code=entry.run(self.root,'developer/policy/GOV/MPD-GOV-0006.yaml',dry_run=True)
        self.assertEqual(2,code)
        self.assertIn('UNREGISTERED_POLICY_IDENTITY',report['problem'])

    def test_reject_authority_escalation(self):
        self.write('.ptsip/contracts/index.json',{'authoring':{'entrypoint':'.ptsip/engine/developer','validators':['policy-rules','responsibility-authority'],'execution_authorized':True}})
        report,code=entry.run(self.root,'developer/policy/GOV/MPD-GOV-0006.yaml',dry_run=True)
        self.assertEqual(2,code)
        self.assertFalse(report['execution_authorized'])

    def test_reject_duplicate_yaml_key(self):
        self.write('developer/policy/GOV/MPD-GOV-0006.yaml','Policy_Identity:\n  root_family: GOV\n  root_family: GOV\n  policy_kind: PTSIP_DEVELOPER_POLICY\n')
        report,code=entry.run(self.root,'developer/policy/GOV/MPD-GOV-0006.yaml',dry_run=True)
        self.assertEqual(2,code)
        self.assertIn('DUPLICATE_YAML_KEY',report['problem'])


if __name__ == '__main__':
    unittest.main()
