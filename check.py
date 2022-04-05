import requests
import sys
import os
import collections

projects = 'http://code.storageos.net/rest/api/1.0/projects/'

username = os.environ.get('USERNAME')
password = os.environ.get('PASSWORD')

response = requests.get(projects, auth=(username, password))

already_migrated = set()
wont_migrate = set()

known_leaky_repos = set()
known_leaky_repos.add("ubuntu")
known_leaky_repos.add("ansible-jenkins-builder-old")
known_leaky_repos.add("api")
known_leaky_repos.add("bats-test")
known_leaky_repos.add("bundle-ingestion")
known_leaky_repos.add("c2")
known_leaky_repos.add("c2-cli")
known_leaky_repos.add("cloud-configs")
known_leaky_repos.add("cloud-spending-reporter")
known_leaky_repos.add("control")
known_leaky_repos.add("coreos")
known_leaky_repos.add("coreos_VM")
known_leaky_repos.add("data")
known_leaky_repos.add("demo-env")
known_leaky_repos.add("dogfood-cluster")
known_leaky_repos.add("drone")
known_leaky_repos.add("edge-deprecated")
known_leaky_repos.add("gke-infra")
known_leaky_repos.add("influxdb")
known_leaky_repos.add("k8s-contrib")
known_leaky_repos.add("keygen-api")
known_leaky_repos.add("kubeadm-provision")
known_leaky_repos.add("load")
known_leaky_repos.add("monitoring")
known_leaky_repos.add("nuodb-ro-repro-cluster")
known_leaky_repos.add("openshift_lab")
known_leaky_repos.add("pagerduty-reports")
known_leaky_repos.add("product-build-e2e-env")
known_leaky_repos.add("product-build-pipeline")
known_leaky_repos.add("proxy")
known_leaky_repos.add("python")
known_leaky_repos.add("python351")
known_leaky_repos.add("rebuild_test")
known_leaky_repos.add("redhat-dpll")
known_leaky_repos.add("registry")
known_leaky_repos.add("rhel8-dpll")
known_leaky_repos.add("st-keygen")
known_leaky_repos.add("t2_aws_test")
known_leaky_repos.add("t2infra")
known_leaky_repos.add("telemetry-ingestor-k8s")
known_leaky_repos.add("terraform")
known_leaky_repos.add("ubuntu")
known_leaky_repos.add("ubuntu-edit")
known_leaky_repos.add("ubuntu-image")
known_leaky_repos.add("ui")
known_leaky_repos.add("ui_STORAGEOS")
known_leaky_repos.add("vol-test")

repo_map = collections.OrderedDict()
leaky_repos = set()

repo_map["cluster_lifecycle"] = "EXISTS"
repo_map["cluster_create_action"] = "EXISTS"
repo_map["demos"] = "EXISTS"
repo_map["docs"] = "EXISTS"
repo_map["documentation"] = "EXISTS"
repo_map["documentation-backend"] = "EXISTS"
repo_map["etcd3-bootstrap"] = "EXISTS"
repo_map["etcd3-terraform"] = "EXISTS"
repo_map["github_actions"] = "EXISTS"
repo_map["github_resources"] = "EXISTS"
repo_map["jenkins-infra"] = "EXISTS"
repo_map["jenkins-vagrant-aws"] = "EXISTS"
repo_map["kubecover"] = "EXISTS"
repo_map["kubecover-actions"] = "EXISTS"
repo_map["kubecover-testsuite"] = "EXISTS"
repo_map["license-api"] = "EXISTS"
repo_map["migrations"] = "EXISTS"
repo_map["opensource-project-template"] = "EXISTS"
repo_map["platform-builder-cli"] = "EXISTS"
repo_map["portal"] = "EXISTS"
repo_map["Instruqt"] = "EXISTS"
repo_map["pre-containers"] = "EXISTS"
repo_map["pre_tests"] = "EXISTS"
repo_map["shared-infrastructure"] = "EXISTS"
repo_map["test-action"] = "EXISTS"
repo_map["test-kubecover"] = "EXISTS"
repo_map["trousseau"] = "EXISTS"
repo_map["use-cases"] = "EXISTS"
repo_map["cluster_lifecycle"] = "EXISTS"
repo_map["metrics-exporter"] = "EXISTS"
repo_map["operator-toolkit"] = "EXISTS"
repo_map["portal-scripts"] = "EXISTS"

def get_repos(username, password, team):
  repos = f'http://code.storageos.net/rest/api/1.0/projects/{team}/repos?limit=1000'

  response = requests.get(repos, auth=(username, password))

  os.system("mkdir -p repos")
  
  # Parse repositories from the JSON
  for repo in response.json()['values']:
    for link in repo['links']['clone']:
      if link['name'] == 'ssh':
        repo_name = repo['slug']
        repo_folder = f'repos/{repo_name}.git'
        repo_description = None
        descr_append = f"Copied from archived bitbucket repo: '{link['href']}' from team '{team}'"

        if (team == 'MIG' or team == 'WON'):
          if (team == 'MIG'):
            print(f"Skipping '{repo_name}' as it is already migrated")
            already_migrated.add(repo_name)
          else:
            print(f"Won't migrate {repo_name} as not required")
            wont_migrate.add(repo_name)
        else:  
          if (repo_name in repo_map):
            repo_name = f'{repo_name}_{team}'
          
          # Use descr, otherwise add old repo
          if ( 'description' in repo):
            repo_map[repo_name]=f"{repo['description']}. {descr_append}" 
          else:  
            repo_map[repo_name]=descr_append
          
          # clone in new otherwise get most recent
          if (os.path.isdir(repo_folder)):
            print(f"folder exists for {repo_name}, updating...")
            os.system(f"cd {repo_folder};git remote update; cd ..")
          else:    
            print(f"folder does not exist for {repo_name}, cloning...")
            os.system(f"cd repos;git clone --mirror {link['href']} {repo_name}")

          # check for leaks
          if (repo_name not in known_leaky_repos):
            has_leaks = os.system(f"gitleaks detect --source {repo_folder}")  
            if (has_leaks):
              print(f"the repo {repo_name} has a leak!!")
              leaky_repos.add(repo_name)

for project in response.json()['values']:
  get_repos(username, password, project['key'])

# grab existing repos
# terraform_existing = open("terraform_repos.json", "r")

# go through list
terraform = open("terraform", "a+")
pre_existing_count = 0
safe_to_fix = set()
for key, value in sorted(repo_map.items()):

  if (key in leaky_repos):
    print(f"the repo {key} has a leak!!")
  else:
    safe_to_fix.add(key)

  if (value == "EXISTS"):
    pre_existing_count = pre_existing_count + 1
  else:  
    if (value):
      line = f'''
      {key} : {{ 
        description = "{value}"
      }}
      '''
    else:
      print("no description for this repo")
      line = f'''
      {key} : {{ }}
      '''
    terraform.write( line )
terraform.close( )

print("These items are safe to migrate:")
for key in safe_to_fix:
  print(f"\t{key}")
print("")

percent_complete = len(already_migrated) + pre_existing_count + len(wont_migrate) / len(repo_map) * 100
print(f"We are {percent_complete}% complete!!")