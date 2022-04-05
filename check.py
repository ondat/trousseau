import requests
import sys
import os
import collections

projects = 'http://code.storageos.net/rest/api/1.0/projects/'

username = os.environ.get('MIG_UNAME')
password = os.environ.get('MIG_PWD')

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
        if (team == 'MIG' or team == 'WON'):
          if (team == 'MIG'):
            print(f"Skipping '{repo_name}' as it is already migrated")
            already_migrated.add(repo_name)
          else:
            print(f"Won't migrate {repo_name} as not required")
            wont_migrate.add(repo_name)
        else:  
          repo_map[repo_name]=team
          
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
          else:
            print(f"the repo {repo_name} has a leak!!")
            leaky_repos.add(repo_name)      

for project in response.json()['values']:
  get_repos(username, password, project['key'])

# go through list
safe_to_fix = open("safe_to_fix.csv", "a+")
pre_existing_count = 0
for key, value in sorted(repo_map.items()):

  if (key in leaky_repos):
    print(f"the repo {key} has a leak!!")
  
  if (key not in leaky_repos and key not in already_migrated and key not in wont_migrate):
    safe_to_fix.write( f'{key}, {value}\n' )

safe_to_fix.close( )

percent_complete = len(already_migrated) + len(wont_migrate) / len(repo_map) * 100
print(f"We are {percent_complete}% complete!!")