FROM python:alpine
COPY main.py main.py
COPY books/ books/
CMD ["python3", "main.py"]
