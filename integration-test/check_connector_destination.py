import sys
import subprocess
import shlex
import csv
import json
import yaml
import jsonschema
import urllib.request
import tempfile


def test_destination_csv():

    tmpCSV = tempfile.NamedTemporaryFile()
    tmpJSONSchema = tempfile.NamedTemporaryFile()

    subprocess.call(shlex.split(
        'docker run -it --privileged --pid=host debian nsenter -t 1 -m -u -n -i sh -c "cat /var/lib/docker/volumes/airbyte/_data/connector-backend-test/_airbyte_raw_detection.csv"'),
        stdout=open(tmpCSV.name, "w"))

    urllib.request.urlretrieve(
        "https://raw.githubusercontent.com/instill-ai/vdp/main/protocol/vdp_protocol.yaml", tmpJSONSchema.name)

    with open(tmpJSONSchema.name) as f:
        jsonSchema = json.loads(json.dumps(yaml.safe_load(f)))

    resolver = jsonschema.RefResolver.from_schema(schema=jsonSchema)

    # read csv file
    with open(tmpCSV.name, encoding="utf-8") as csvf:
        # load csv file data using csv library's dictionary reader
        csvReader = csv.DictReader(csvf)

        # convert each csv row into python dict
        for row in csvReader:
            try:
                jsonschema.validate(
                    instance=json.loads(row["_airbyte_data"]),
                    schema={"$ref": "#/definitions/TaskOutput"},
                    resolver=resolver,
                    cls=jsonschema.Draft7Validator,
                )
            except:
                sys.exit(1)

    sys.exit(0)


if __name__ == "__main__":
    test_destination_csv()
