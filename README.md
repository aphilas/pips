# pips

Tiny tool to install PyPi packages with `pip` and update `requirements.txt`.

If you're lost, you are probably looking for [Poetry](https://python-poetry.org/), [pipenv](https://pipenv.pypa.io/en/latest/index.html), or [PDM](https://pdm.fming.dev/latest/).

## Installation

### Build from source

```sh
git clone https://github.com/aphilas/pips.git
cd pips
go build -o /usr/local/bin/pips .
pips --help
```

## Usage

```sh
cd /path/to/project

python -m venv venv
source venv/bin/activate

pips install starlette[full]
cat requirements.txt
# starlette[full]==3.6.2

pips uninstall starlette
cat requirements.txt
# 
```

## Motivation

I don't (yet) need Poetry. I had a [bash function](https://gist.githubusercontent.com/aphilas/6bf28a7bb71a66f2a974d27e1ca3ff30/raw/8df298268915294746e19e918ee2becf62586743/pips.sh) to do this that was getting unwieldy.
